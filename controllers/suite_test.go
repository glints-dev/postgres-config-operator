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
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

var postgresContainer testcontainers.Container
var postgresConn *pgx.Conn

const postgresUser = "glints"
const postgresPassword = "glints"
const postgresSecretName = "postgres"

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = postgresv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

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

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// SetupPostgresContainer creates a PostgreSQL container for testing purposes.
// The created container is accessible through the postgresContainer global.
func SetupPostgresContainer(ctx context.Context) {
	BeforeEach(func() {
		const port = "5432/tcp"
		req := testcontainers.ContainerRequest{
			Image: "postgres:12-alpine",
			Env: map[string]string{
				"POSTGRES_USER":     postgresUser,
				"POSTGRES_PASSWORD": postgresPassword,
			},
			ExposedPorts: []string{port},
			WaitingFor:   wait.ForListeningPort(port),
		}

		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		Expect(err).NotTo(HaveOccurred())

		retVal, err := container.Exec(ctx, []string{
			"psql", "-U", postgresUser, "-c", "CREATE EXTENSION pgcrypto;",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(retVal).To(Equal(0))

		postgresContainer = container
	})

	AfterEach(func() {
		postgresContainer.Terminate(ctx)
	})
}

// SetupPostgresConnection creates a connection to the test PostgreSQL
// container. This must be run after SetupPostgresContainer.
func SetupPostgresConnection(ctx context.Context) {
	BeforeEach(func() {
		endpoint, err := postgresContainer.Endpoint(ctx, "")
		Expect(err).NotTo(HaveOccurred())

		conn, err := pgx.Connect(ctx, fmt.Sprintf(
			"postgres://%s:%s@%s/glints",
			postgresUser,
			postgresPassword,
			endpoint,
		))
		Expect(err).NotTo(HaveOccurred())

		postgresConn = conn
	})

	AfterEach(func() {
		postgresConn.Close(ctx)
	})
}

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
	err := k8sClient.Delete(ctx, ns)
	Expect(err).NotTo(HaveOccurred())
}

// PostgresContainerRef returns a reference to the test Postgres container,
// which can be used within a PostgresConfig resource.
func PostgresContainerRef(ctx context.Context) postgresv1alpha1.PostgresRef {
	endpoint, err := postgresContainer.Endpoint(ctx, "")
	Expect(err).NotTo(HaveOccurred())

	hostPort := strings.SplitN(endpoint, ":", 2)
	port, err := strconv.Atoi(hostPort[1])
	Expect(err).NotTo(HaveOccurred())

	return postgresv1alpha1.PostgresRef{
		Host:     hostPort[0],
		Port:     uint16(port),
		Database: "glints",
		SecretRef: postgresv1alpha1.SecretRef{
			SecretName: postgresSecretName,
		},
	}
}

// CreatePostgresSecret creates a Secret that's pre-configured to connect to the
// test PostgreSQL server.
func CreatePostgresSecret(ctx context.Context, namespace string) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      postgresSecretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"POSTGRES_USER":     postgresUser,
			"POSTGRES_PASSWORD": postgresPassword,
		},
	}

	err := k8sClient.Create(ctx, secret)
	Expect(err).NotTo(HaveOccurred(), "failed to create test Secret resource")
}