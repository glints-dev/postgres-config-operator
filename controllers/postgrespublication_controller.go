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
	"net/url"

	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
	"github.com/glints-dev/postgres-config-operator/controllers/publicationmanager"
	"github.com/glints-dev/postgres-config-operator/controllers/utils"
)

const (
	EventTypeInvalidVariant string = "InvalidVariant"
)

const (
	VariantStandard string = "standard"
	VariantAiven    string = "aiven"
)

// PostgresPublicationReconciler reconciles a PostgresPublication object
type PostgresPublicationReconciler struct {
	client.Client
	recorder record.EventRecorder
	Log      logr.Logger
	Scheme   *runtime.Scheme
}

//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgrespublications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgrespublications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgrespublications/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *PostgresPublicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = utils.WithRequestLogger(ctx, req, "postgrespublication", r.Log)

	publication := &postgresv1alpha1.PostgresPublication{}
	if err := r.Get(ctx, req.NamespacedName, publication); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	secretNamespacedName := types.NamespacedName{
		Name:      publication.Spec.PostgresRef.SecretRef.SecretName,
		Namespace: publication.GetNamespace(),
	}
	secretRef := &corev1.Secret{}
	if err := r.Get(ctx, secretNamespacedName, secretRef); err != nil {
		r.recorder.Eventf(
			publication,
			corev1.EventTypeWarning,
			EventTypeMissingSecret,
			"failed to get Secret: %v",
			err,
		)
		return ctrl.Result{Requeue: true}, nil
	}

	connURL := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			string(secretRef.Data["POSTGRES_USER"]),
			string(secretRef.Data["POSTGRES_PASSWORD"]),
		),
		Host: fmt.Sprintf(
			"%s:%d",
			publication.Spec.PostgresRef.Host,
			publication.Spec.PostgresRef.Port,
		),
		RawPath: publication.Spec.PostgresRef.Database,
	}
	conn, err := pgx.Connect(ctx, connURL.String())
	if err != nil {
		r.recorder.Eventf(
			publication,
			corev1.EventTypeWarning,
			EventTypeFailedConnectPostgres,
			"failed to connect to PostgreSQL: %v",
			err,
		)
		return ctrl.Result{Requeue: true}, nil
	}
	defer conn.Close(ctx)

	publicationManager, err := r.publicationManager(publication, conn)
	if err != nil {
		r.recorder.Eventf(
			publication,
			corev1.EventTypeWarning,
			EventTypeInvalidVariant,
			"invalid variant configured: %v",
			err,
		)
		return ctrl.Result{Requeue: true}, nil
	}

	r.recorder.Event(
		publication,
		corev1.EventTypeNormal,
		EventTypeSuccessConnectPostgres,
		"successfully connected to PostgreSQL",
	)

	if stopReconcilation, err := r.handleFinalizers(
		ctx,
		publicationManager,
		publication,
	); err != nil || stopReconcilation {
		return ctrl.Result{}, err
	}

	reconcileResult, err := r.reconcile(ctx, publicationManager, publication)
	if err != nil {
		return reconcileResult, fmt.Errorf("failed to reconcile: %w", err)
	}

	if reconcileResult.Requeue {
		return reconcileResult, nil
	}

	return ctrl.Result{}, nil
}

func (r *PostgresPublicationReconciler) publicationManager(
	publication *postgresv1alpha1.PostgresPublication,
	conn *pgx.Conn,
) (publicationmanager.Manager, error) {
	switch publication.Spec.PostgresRef.Variant {
	case VariantStandard:
		fallthrough
	case "":
		return &publicationmanager.StdManager{Conn: conn}, nil
	case VariantAiven:
		return &publicationmanager.AivenManager{
			StdManager: publicationmanager.StdManager{Conn: conn},
		}, nil
	}

	return nil, fmt.Errorf("unrecognized publication manager \"%s\"", publication.Spec.PostgresRef.Variant)
}

// handleFinalizers implements the logic required to handle deletes.
func (r *PostgresPublicationReconciler) handleFinalizers(
	ctx context.Context,
	manager publicationmanager.Manager,
	publication *postgresv1alpha1.PostgresPublication,
) (stopReconcilation bool, err error) {
	const finalizerName = "postgres.glints.com/finalizer"

	if publication.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted.
		if !containsString(publication.GetFinalizers(), finalizerName) {
			controllerutil.AddFinalizer(publication, finalizerName)
			if err := r.Update(ctx, publication); err != nil {
				return false, err
			}
		}
	} else {
		// The object is being deleted.
		if containsString(publication.GetFinalizers(), finalizerName) {
			if err := r.deleteExternalResources(ctx, manager, publication); err != nil {
				return false, err
			}

			controllerutil.RemoveFinalizer(publication, finalizerName)
			if err := r.Update(ctx, publication); err != nil {
				return false, err
			}
		}

		return true, nil
	}

	return false, nil
}

// deleteExternalResources removes all resources associated with the given
// PostgresPublication.
func (r *PostgresPublicationReconciler) deleteExternalResources(
	ctx context.Context,
	manager publicationmanager.Manager,
	publication *postgresv1alpha1.PostgresPublication,
) error {
	utils.RequestLogger(ctx, r.Log).Info("dropping publication", "name", publication.Spec.Name)

	if err := manager.DropPublication(ctx, publication.Spec.Name); err != nil {
		return fmt.Errorf("failed to drop publication: %w", err)
	}

	return nil
}

// reconcile performs the reconcillation with the given connection.
func (r *PostgresPublicationReconciler) reconcile(
	ctx context.Context,
	manager publicationmanager.Manager,
	publication *postgresv1alpha1.PostgresPublication,
) (ctrl.Result, error) {
	r.Log.Info("reconcilling publication", "publication.Name", publication.Name)
	var err error

	created, err := manager.CreatePublication(ctx, publication)
	if err != nil {
		r.recorder.Eventf(
			publication,
			corev1.EventTypeWarning,
			EventTypeFailedReconcile,
			"failed to create publication: %v",
			err,
		)
		return ctrl.Result{Requeue: true}, nil
	}

	if !created {
		if err := manager.AlterExistingPublication(ctx, publication); err != nil {
			r.recorder.Eventf(
				publication,
				corev1.EventTypeWarning,
				EventTypeFailedReconcile,
				"failed to alter existing publication: %v",
				err,
			)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PostgresPublicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("postgres-publication-controller")

	return ctrl.NewControllerManagedBy(mgr).
		For(&postgresv1alpha1.PostgresPublication{}).
		Complete(r)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
