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
	"strings"

	"github.com/go-logr/logr"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
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
	_ = r.Log.WithValues("postgrespublication", req.NamespacedName)

	publication := &postgresv1alpha1.PostgresPublication{}
	if err := r.Get(ctx, req.NamespacedName, publication); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get PostgresPublication: %w", err)
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

	r.recorder.Event(
		publication,
		corev1.EventTypeNormal,
		EventTypeSuccessConnectPostgres,
		"successfully connected to PostgreSQL",
	)

	if stopReconcilation, err := r.handleFinalizers(ctx, conn, publication); err != nil || stopReconcilation {
		return ctrl.Result{}, err
	}

	reconcileResult, err := r.reconcile(ctx, conn, publication)
	if err != nil {
		return reconcileResult, fmt.Errorf("failed to reconcile: %w", err)
	}

	if reconcileResult.Requeue {
		return reconcileResult, nil
	}

	return ctrl.Result{}, nil
}

// handleFinalizers implements the logic required to handle deletes.
func (r *PostgresPublicationReconciler) handleFinalizers(
	ctx context.Context,
	conn *pgx.Conn,
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
			if err := r.deleteExternalResources(ctx, conn, publication); err != nil {
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
	conn *pgx.Conn,
	publication *postgresv1alpha1.PostgresPublication,
) error {
	query := fmt.Sprintf(
		"DROP PUBLICATION %s",
		pgx.Identifier{publication.Spec.Name}.Sanitize(),
	)

	if _, err := conn.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to drop publication: %w", err)
	}

	return nil
}

// reconcile performs the reconcillation with the given connection.
func (r *PostgresPublicationReconciler) reconcile(
	ctx context.Context,
	conn *pgx.Conn,
	publication *postgresv1alpha1.PostgresPublication,
) (ctrl.Result, error) {
	r.Log.Info("reconcilling publication")
	var err error

	query, err := r.buildCreatePublicationQuery(publication, conn.PgConn())
	if err != nil {
		r.recorder.Eventf(
			publication,
			corev1.EventTypeWarning,
			EventTypeFailedReconcile,
			"failed to build create publication query: %v",
			err,
		)
		return ctrl.Result{Requeue: true}, nil
	}

	r.Log.Info(fmt.Sprintf("executing query: %s", query))
	_, err = conn.Exec(ctx, query)

	publicationCreated := true
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok && pgErr.Code == pgerrcode.DuplicateObject {
			publicationCreated = false
		} else {
			r.recorder.Eventf(
				publication,
				corev1.EventTypeWarning,
				EventTypeFailedReconcile,
				"failed to execute query: %v",
				err,
			)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	if !publicationCreated {
		if err := r.alterExistingPublication(ctx, conn, publication); err != nil {
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

// ReconcilePublication builds the query to create a publication.
func (r *PostgresPublicationReconciler) buildCreatePublicationQuery(
	publication *postgresv1alpha1.PostgresPublication,
	conn *pgconn.PgConn,
) (string, error) {
	publicationIdentifer := pgx.Identifier{publication.Spec.Name}

	var forTablePart string
	if len(publication.Spec.Tables) == 0 {
		forTablePart = "FOR ALL TABLES"
	} else {
		var tableIdentifiers []string
		for _, table := range publication.Spec.Tables {
			tableIdentifier := pgx.Identifier{table.Schema, table.Name}
			tableIdentifiers = append(tableIdentifiers, tableIdentifier.Sanitize())
		}

		forTablePart = fmt.Sprintf("FOR TABLE %s", strings.Join(tableIdentifiers, ", "))
	}

	var withPublicationParameterPart string
	if len(publication.Spec.Operations) > 0 {
		escapedOperations, err := conn.EscapeString(strings.Join(publication.Spec.Operations, ", "))
		if err != nil {
			return "", fmt.Errorf("failed to escape string: %w", err)
		}

		withPublicationParameterPart = fmt.Sprintf(
			"WITH (publish = '%s')",
			escapedOperations,
		)
	}

	return strings.Join(
		[]string{
			"CREATE PUBLICATION",
			publicationIdentifer.Sanitize(),
			forTablePart,
			withPublicationParameterPart,
		},
		" ",
	), nil
}

// alterExistingPublication modifies an existing publication to the desired
// state as described in publication.
func (r *PostgresPublicationReconciler) alterExistingPublication(
	ctx context.Context,
	conn *pgx.Conn,
	publication *postgresv1alpha1.PostgresPublication,
) error {
	var tableIdentifiers []string
	for _, table := range publication.Spec.Tables {
		tableIdentifier := pgx.Identifier{table.Schema, table.Name}
		tableIdentifiers = append(tableIdentifiers, tableIdentifier.Sanitize())
	}

	setTableQuery := fmt.Sprintf(
		"ALTER PUBLICATION %s SET TABLE %s",
		pgx.Identifier{publication.Spec.Name}.Sanitize(),
		strings.Join(tableIdentifiers, ", "),
	)

	r.Log.Info(fmt.Sprintf("executing query: %s", setTableQuery))
	if _, err := conn.Exec(ctx, setTableQuery); err != nil {
		return fmt.Errorf("failed to set table to publication: %w", err)
	}

	var joinedOperations string
	if len(publication.Spec.Operations) > 0 {
		joinedOperations = strings.Join(publication.Spec.Operations, ", ")
	} else {
		// Default value as documented here:
		// https://www.postgresql.org/docs/current/sql-createpublication.html
		joinedOperations = "insert, update, delete, truncate"
	}

	sanitizedOperations, err := conn.PgConn().EscapeString(joinedOperations)
	if err != nil {
		return fmt.Errorf("failed to escape operations: %w", err)
	}

	setParamQuery := fmt.Sprintf(
		"ALTER PUBLICATION %s SET (publish = '%s')",
		pgx.Identifier{publication.Spec.Name}.Sanitize(),
		sanitizedOperations,
	)

	r.Log.Info(fmt.Sprintf("executing query: %s", setParamQuery))
	if _, err := conn.Exec(
		ctx,
		setParamQuery,
	); err != nil {
		return fmt.Errorf("failed to set publication parameter: %w", err)
	}

	return nil
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
