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
	"strings"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
)

const (
	EventTypeMissingSecret          string = "MissingSecret"
	EventTypeFailedConnectPostgres  string = "FailedConnectPostgres"
	EventTypeSuccessConnectPostgres string = "SuccessConnectPostgres"
	EventTypeFailedReconcile        string = "FailedReconcile"
	EventTypeSuccessfulReconcile    string = "SuccessfulReconcile"
)

var (
	publicationOwnerKey string = ".metadata.controller"
	apiGVStr            string = postgresv1alpha1.GroupVersion.String()
)

// PostgresConfigReconciler reconciles a PostgresConfig object
type PostgresConfigReconciler struct {
	client.Client
	recorder record.EventRecorder
	Log      logr.Logger
	Scheme   *runtime.Scheme
}

//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgresconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgresconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgresconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *PostgresConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("postgresconfig", req.NamespacedName)

	postgresConfig := &postgresv1alpha1.PostgresConfig{}
	if err := r.Get(ctx, req.NamespacedName, postgresConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var currentPublications postgresv1alpha1.PostgresPublicationList
	if err := r.List(
		ctx,
		&currentPublications,
		&client.ListOptions{
			FieldSelector: fields.SelectorFromSet(fields.Set{publicationOwnerKey: req.Name}),
			Namespace:     req.Namespace,
		},
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to list owned publications: %w", err)
	}

	currentPublicationsByName := make(map[string]postgresv1alpha1.PostgresPublication)
	for _, publication := range currentPublications.Items {
		currentPublicationsByName[publication.Spec.Name] = publication
	}

	desiredPublicationsByName := make(map[string]postgresv1alpha1.Publication)
	for _, publication := range postgresConfig.Spec.Publications {
		desiredPublicationsByName[publication.Name] = publication
	}

	// Compare the current and desired list of publications.
	var publicationsToCreate []postgresv1alpha1.Publication
	var publicationsToUpdate []postgresv1alpha1.Publication
	for name, publication := range desiredPublicationsByName {
		if _, ok := currentPublicationsByName[name]; !ok {
			publicationsToCreate = append(publicationsToCreate, publication)
		} else {
			publicationsToUpdate = append(publicationsToUpdate, publication)
		}
	}

	var publicationsToDelete []postgresv1alpha1.PostgresPublication
	for name, publication := range currentPublicationsByName {
		if _, ok := desiredPublicationsByName[name]; !ok {
			publicationsToDelete = append(publicationsToDelete, publication)
		}
	}

	if err := r.createPublications(
		ctx,
		*postgresConfig,
		publicationsToCreate,
		req.Name,
		req.Namespace,
		postgresConfig.Spec.PostgresRef,
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create publication resources: %w", err)
	}

	if err := r.updatePublications(
		ctx,
		*postgresConfig,
		publicationsToCreate,
		req.Name,
		req.Namespace,
		postgresConfig.Spec.PostgresRef,
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update publication resources: %w", err)
	}

	if err := r.deletePublications(
		ctx,
		publicationsToDelete,
		req.Name,
		req.Namespace,
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to delete publication resources: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *PostgresConfigReconciler) createPublications(
	ctx context.Context,
	config postgresv1alpha1.PostgresConfig,
	publications []postgresv1alpha1.Publication,
	configName string,
	namespace string,
	postgresRef postgresv1alpha1.PostgresRef,
) error {
	for _, publication := range publications {
		resource := r.buildPublicationResource(
			config,
			publication,
			configName,
			namespace,
			postgresRef,
		)

		if err := r.Create(ctx, resource); err != nil {
			return fmt.Errorf("failed to create publication resource: %w", err)
		}
	}

	return nil
}

func (r *PostgresConfigReconciler) updatePublications(
	ctx context.Context,
	config postgresv1alpha1.PostgresConfig,
	publications []postgresv1alpha1.Publication,
	configName string,
	namespace string,
	postgresRef postgresv1alpha1.PostgresRef,
) error {
	for _, publication := range publications {
		resource := r.buildPublicationResource(
			config,
			publication,
			configName,
			namespace,
			postgresRef,
		)

		if err := r.Update(ctx, resource); err != nil {
			return fmt.Errorf("failed to update publication resource: %w", err)
		}
	}

	return nil
}

func (r *PostgresConfigReconciler) buildPublicationResource(
	config postgresv1alpha1.PostgresConfig,
	publication postgresv1alpha1.Publication,
	configName string,
	namespace string,
	postgresRef postgresv1alpha1.PostgresRef,
) *postgresv1alpha1.PostgresPublication {
	return &postgresv1alpha1.PostgresPublication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.publicationResourceName(configName, publication),
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(config.GetObjectMeta(), config.GroupVersionKind()),
			},
		},
		Spec: postgresv1alpha1.PostgresPublicationSpec{
			PostgresRef: postgresRef,
			Name:        publication.Name,
			Tables:      publication.Tables,
			Operations:  publication.Operations,
		},
	}
}

func (r *PostgresConfigReconciler) deletePublications(
	ctx context.Context,
	publications []postgresv1alpha1.PostgresPublication,
	configName string,
	namespace string,
) error {
	for _, publication := range publications {
		if err := r.Delete(ctx, &publication); err != nil {
			return fmt.Errorf("failed to delete publication resource: %w", err)
		}
	}

	return nil
}

func (r *PostgresConfigReconciler) publicationResourceName(
	configName string,
	publication postgresv1alpha1.Publication,
) string {
	return fmt.Sprintf(
		"%s-%s",
		configName,
		strings.ReplaceAll(publication.Name, "_", "-"),
	)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PostgresConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("postgres-config-controller")

	// Index the owner of child PostgresPublications to allow for fast lookups.
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&postgresv1alpha1.PostgresPublication{},
		publicationOwnerKey,
		func(rawObj client.Object) []string {
			publication := rawObj.(*postgresv1alpha1.PostgresPublication)
			owner := metav1.GetControllerOf(publication)
			if owner == nil {
				return nil
			}

			if owner.APIVersion != apiGVStr || owner.Kind != "PostgresConfig" {
				return nil
			}

			return []string{owner.Name}
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&postgresv1alpha1.PostgresConfig{}).
		Owns(&postgresv1alpha1.PostgresPublication{}).
		Complete(r)
}
