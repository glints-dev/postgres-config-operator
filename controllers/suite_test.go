/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
	"github.com/glints-dev/postgres-config-operator/controllers/testutils"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = postgresv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&PostgresConfigReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("postgresconfig"),
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	err = (&PostgresPublicationReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("postgrespublication"),
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	err = (&PostgresTableReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("postgrestable"),
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()
}, 60)

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// CreateTestNamespace creates an independent namespace for testing.
func CreateTestNamespace(ctx context.Context) *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}

	err := k8sClient.Create(ctx, namespace)
	Expect(err).NotTo(HaveOccurred())

	return namespace
}

// DeleteNamespace deletes the given namespace.
func DeleteNamespace(ctx context.Context, ns *corev1.Namespace) {
	// Use the Kubernetes Discovery API to look up all namespaced resources. The
	// standard Kubernetes client is used instead of the controller-runtime one
	// because the latter doesn't support the Discovery API.
	stdK8sClient, err := kubernetes.NewForConfig(testEnv.Config)
	Expect(err).NotTo(HaveOccurred())

	_, apiResources, err := stdK8sClient.Discovery().ServerGroupsAndResources()
	Expect(err).NotTo(HaveOccurred())

	// Get all namespaced GroupVersionKinds.
	namespacedGVKs := make(map[string]schema.GroupVersionKind)
	for _, apiResourceList := range apiResources {
		groupVersion, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		Expect(err).NotTo(HaveOccurred())

		for _, resource := range apiResourceList.APIResources {
			if !resource.Namespaced || strings.Contains(resource.Name, "/") {
				continue
			}

			gvk := schema.GroupVersionKind{
				Group:   groupVersion.Group,
				Version: groupVersion.Version,
				Kind:    resource.Kind,
			}

			if resource.Group != "" {
				gvk.Group = resource.Group
			}

			if resource.Version != "" {
				gvk.Version = resource.Version
			}

			namespacedGVKs[gvk.String()] = gvk
		}
	}

	// Delete all namespaced resources.
	for _, gvk := range namespacedGVKs {
		var u unstructured.Unstructured
		u.SetGroupVersionKind(gvk)

		err := k8sClient.DeleteAllOf(ctx, &u, client.InNamespace(ns.Name))
		Expect(client.IgnoreNotFound(ignoreMethodNotAllowed(err))).ShouldNot(HaveOccurred())
	}

	// Remove the namespace finalizer.
	Eventually(func() error {
		key := client.ObjectKeyFromObject(ns)

		if err := k8sClient.Get(ctx, key, ns); err != nil {
			return client.IgnoreNotFound(err)
		}

		finalizers := []corev1.FinalizerName{}
		for _, finalizer := range ns.Spec.Finalizers {
			if finalizer != "kubernetes" {
				finalizers = append(finalizers, finalizer)
			}
		}

		ns.Spec.Finalizers = finalizers

		_, err = stdK8sClient.CoreV1().Namespaces().Finalize(ctx, ns, metav1.UpdateOptions{})
		return err
	}).Should(Succeed())

	// Finally delete the namespace.
	err = k8sClient.Delete(ctx, ns)
	Expect(err).NotTo(HaveOccurred())
}

func ignoreMethodNotAllowed(err error) error {
	if err != nil {
		if apierrors.ReasonForError(err) == metav1.StatusReasonMethodNotAllowed {
			return nil
		}
	}
	return err
}

// CreatePostgresSecret creates a Secret that's pre-configured to connect to the
// test PostgreSQL server.
func CreatePostgresSecret(ctx context.Context, namespace string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testutils.PostgresSecretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"POSTGRES_USER":     testutils.PostgresUser,
			"POSTGRES_PASSWORD": testutils.PostgresPassword,
		},
	}

	err := k8sClient.Create(ctx, secret)
	Expect(err).NotTo(HaveOccurred(), "failed to create test Secret resource")
}

// createBarebonesTable creates a very minimal table with the given name.
func createBarebonesTable(ctx context.Context, name string) {
	query := fmt.Sprintf(
		`CREATE TABLE %s (
			id UUID NOT NULL DEFAULT gen_random_uuid(),
			CONSTRAINT %s PRIMARY KEY (id)
		)`,
		pgx.Identifier{name}.Sanitize(),
		pgx.Identifier{fmt.Sprintf("%s_pkey", name)}.Sanitize(),
	)

	_, err := testutils.PostgresConn.Exec(ctx, query)
	Expect(err).NotTo(HaveOccurred(), "failed to create table %s", name)
}
