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

const (
	EventTypeMissingSecret          string = "MissingSecret"
	EventTypeFailedConnectPostgres  string = "FailedConnectPostgres"
	EventTypeSuccessConnectPostgres string = "SuccessConnectPostgres"
	EventTypeFailedReconcile        string = "FailedReconcile"
	EventTypeSuccessfulReconcile    string = "SuccessfulReconcile"
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
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *PostgresConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("postgresconfig", req.NamespacedName)

	postgresConfig := &postgresv1alpha1.PostgresConfig{}
	if err := r.Get(ctx, req.NamespacedName, postgresConfig); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get PostgresConfig: %w", err)
	}

	secretNamespacedName := types.NamespacedName{
		Name:      postgresConfig.Spec.PostgresRef.SecretRef.SecretName,
		Namespace: postgresConfig.GetNamespace(),
	}
	secretRef := &corev1.Secret{}
	if err := r.Get(ctx, secretNamespacedName, secretRef); err != nil {
		r.recorder.Eventf(
			postgresConfig,
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
			postgresConfig.Spec.PostgresRef.Host,
			postgresConfig.Spec.PostgresRef.Port,
		),
		RawPath: postgresConfig.Spec.PostgresRef.Database,
	}
	conn, err := pgx.Connect(ctx, connURL.String())
	if err != nil {
		r.recorder.Eventf(
			postgresConfig,
			corev1.EventTypeWarning,
			EventTypeFailedConnectPostgres,
			"failed to connect to PostgreSQL: %v",
			err,
		)
		return ctrl.Result{Requeue: true}, nil
	}
	defer conn.Close(ctx)

	r.recorder.Event(
		postgresConfig,
		corev1.EventTypeNormal,
		EventTypeSuccessConnectPostgres,
		"successfully connected to PostgreSQL",
	)

	if err := r.HandleFinalizers(ctx, conn, postgresConfig); err != nil {
		return ctrl.Result{}, err
	}

	reconcileResult := r.reconcileWithConnAndConfig(ctx, conn, postgresConfig)
	if reconcileResult.Requeue {
		return reconcileResult, nil
	}

	return ctrl.Result{}, nil
}

// HandleFinalizers implements the logic required to handle deletes.
func (r *PostgresConfigReconciler) HandleFinalizers(
	ctx context.Context,
	conn *pgx.Conn,
	config *postgresv1alpha1.PostgresConfig,
) error {
	const finalizerName = "postgres.glints.com/finalizer"

	if config.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted.
		if !containsString(config.GetFinalizers(), finalizerName) {
			controllerutil.AddFinalizer(config, finalizerName)
			if err := r.Update(ctx, config); err != nil {
				return err
			}
		}
	} else {
		// The object is being deleted.
		if containsString(config.GetFinalizers(), finalizerName) {
			if err := r.deleteExternalResources(ctx, conn, config); err != nil {
				return err
			}

			controllerutil.RemoveFinalizer(config, finalizerName)
			if err := r.Update(ctx, config); err != nil {
				return err
			}
		}
	}

	return nil
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// deleteExternalResources removes all resources associated with the given
// PostgresConfig. Note that this will also remove resources not originally
// managed by the operator, so use with care.
func (r *PostgresConfigReconciler) deleteExternalResources(
	ctx context.Context,
	conn *pgx.Conn,
	config *postgresv1alpha1.PostgresConfig,
) error {
	for _, publication := range config.Spec.Publications {
		query := fmt.Sprintf(
			"DROP PUBLICATION %s",
			pgx.Identifier{publication.Name}.Sanitize(),
		)

		if _, err := conn.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to drop publication: %w", err)
		}
	}

	return nil
}

// reconcileWithConnAndConfig performs the reconcillation with the given
// connection and configuration.
func (r *PostgresConfigReconciler) reconcileWithConnAndConfig(
	ctx context.Context,
	conn *pgx.Conn,
	config *postgresv1alpha1.PostgresConfig,
) ctrl.Result {
	r.Log.Info("reconcilling publications")

	if err := r.reconcilePublications(
		ctx,
		conn,
		config.Spec.Publications,
	); err != nil {
		r.recorder.Eventf(
			config,
			corev1.EventTypeWarning,
			EventTypeFailedReconcile,
			"failed to reconcile: %v",
			err,
		)
		return ctrl.Result{Requeue: true}
	}

	r.Log.Info("publications reconcilled successfully")
	r.recorder.Event(
		config,
		corev1.EventTypeNormal,
		EventTypeSuccessfulReconcile,
		"configuration reconcilled successfully",
	)

	return ctrl.Result{}
}

// reconcilePublications ensures that publications on the PostgreSQL server are
// consistent with the given configuration.
func (r *PostgresConfigReconciler) reconcilePublications(
	ctx context.Context,
	conn *pgx.Conn,
	publications []postgresv1alpha1.PostgresPublication,
) error {
	for _, publication := range publications {
		if err := r.reconcilePublication(ctx, conn, publication); err != nil {
			return fmt.Errorf("failed to reconcile publication: %w", err)
		}
	}

	return nil
}

// reconcilePublication ensures that the given publication is reconciled.
func (r *PostgresConfigReconciler) reconcilePublication(
	ctx context.Context,
	conn *pgx.Conn,
	publication postgresv1alpha1.PostgresPublication,
) error {
	var err error

	query, err := r.buildCreatePublicationQuery(publication, conn.PgConn())
	if err != nil {
		return fmt.Errorf("failed to build create publication query: %w", err)
	}

	r.Log.Info(fmt.Sprintf("executing query: %s", query))
	_, err = conn.Exec(ctx, query)

	publicationCreated := true
	if err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if ok && pgErr.Code == pgerrcode.DuplicateObject {
			publicationCreated = false
		} else {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	if !publicationCreated {
		if err := r.alterExistingPublication(ctx, conn, publication); err != nil {
			return fmt.Errorf("failed to alter existing publication: %w", err)
		}
	}

	return nil
}

// ReconcilePublication builds the query to create a publication.
func (r *PostgresConfigReconciler) buildCreatePublicationQuery(
	publication postgresv1alpha1.PostgresPublication,
	conn *pgconn.PgConn,
) (string, error) {
	publicationIdentifer := pgx.Identifier{publication.Name}

	var forTablePart string
	if len(publication.Tables) == 0 {
		forTablePart = "FOR ALL TABLES"
	} else {
		var tableIdentifiers []string
		for _, table := range publication.Tables {
			tableIdentifier := pgx.Identifier{table.Schema, table.Name}
			tableIdentifiers = append(tableIdentifiers, tableIdentifier.Sanitize())
		}

		forTablePart = fmt.Sprintf("FOR TABLE %s", strings.Join(tableIdentifiers, ", "))
	}

	var withPublicationParameterPart string
	if len(publication.Operations) > 0 {
		escapedOperations, err := conn.EscapeString(strings.Join(publication.Operations, ", "))
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
func (r *PostgresConfigReconciler) alterExistingPublication(
	ctx context.Context,
	conn *pgx.Conn,
	publication postgresv1alpha1.PostgresPublication,
) error {
	var tableIdentifiers []string
	for _, table := range publication.Tables {
		tableIdentifier := pgx.Identifier{table.Schema, table.Name}
		tableIdentifiers = append(tableIdentifiers, tableIdentifier.Sanitize())
	}

	setTableQuery := fmt.Sprintf(
		"ALTER PUBLICATION %s SET TABLE %s",
		pgx.Identifier{publication.Name}.Sanitize(),
		strings.Join(tableIdentifiers, ", "),
	)

	r.Log.Info(fmt.Sprintf("executing query: %s", setTableQuery))
	if _, err := conn.Exec(ctx, setTableQuery); err != nil {
		return fmt.Errorf("failed to set table to publication: %w", err)
	}

	var joinedOperations string
	if len(publication.Operations) > 0 {
		joinedOperations = strings.Join(publication.Operations, ", ")
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
		pgx.Identifier{publication.Name}.Sanitize(),
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
func (r *PostgresConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("postgres-config-controller")

	return ctrl.NewControllerManagedBy(mgr).
		For(&postgresv1alpha1.PostgresConfig{}).
		Complete(r)
}
