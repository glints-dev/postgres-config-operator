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

	reconcileResult := r.ReconcileWithConnAndConfig(ctx, conn, postgresConfig)
	if reconcileResult.Requeue {
		return reconcileResult, nil
	}

	return ctrl.Result{}, nil
}

// ReconcileWithConnAndConfig performs the reconcillation with the given
// connection and configuration.
func (r *PostgresConfigReconciler) ReconcileWithConnAndConfig(
	ctx context.Context,
	conn *pgx.Conn,
	config *postgresv1alpha1.PostgresConfig,
) ctrl.Result {
	r.Log.Info("reconcilling publications")

	if err := r.ReconcilePublications(
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

// ReconcilePublications ensures that publications on the PostgreSQL server are
// consistent with the given configuration.
func (r *PostgresConfigReconciler) ReconcilePublications(
	ctx context.Context,
	conn *pgx.Conn,
	publications []postgresv1alpha1.PostgresPublication,
) error {
	for _, publication := range publications {
		if err := r.ReconcilePublication(ctx, conn, publication); err != nil {
			return fmt.Errorf("failed to reconcile publication: %w", err)
		}
	}

	return nil
}

// ReconcilePublication ensures that the given publication is reconciled.
func (r *PostgresConfigReconciler) ReconcilePublication(
	ctx context.Context,
	conn *pgx.Conn,
	publication postgresv1alpha1.PostgresPublication,
) error {
	var err error

	query := r.BuildCreatePublicationQuery(publication)
	r.Log.Info(fmt.Sprintf("executing query: %s", query))
	if len(publication.Operations) > 0 {
		_, err = conn.Exec(ctx, query, strings.Join(publication.Operations, ", "))
	} else {
		_, err = conn.Exec(ctx, query)
	}

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
		if err := r.AlterExistingPublication(ctx, conn, publication); err != nil {
			return fmt.Errorf("failed to alter existing publication: %w", err)
		}
	}

	return nil
}

// ReconcilePublication builds the query to create a publication.
func (r *PostgresConfigReconciler) BuildCreatePublicationQuery(
	publication postgresv1alpha1.PostgresPublication,
) string {
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
		withPublicationParameterPart = fmt.Sprintf("WITH (publish = $1)")
	}

	return strings.Join(
		[]string{
			"CREATE PUBLICATION",
			publicationIdentifer.Sanitize(),
			forTablePart,
			withPublicationParameterPart,
		},
		" ",
	)
}

// AlterExistingPublication modifies an existing publication to the desired
// state as described in publication.
func (r *PostgresConfigReconciler) AlterExistingPublication(
	ctx context.Context,
	conn *pgx.Conn,
	publication postgresv1alpha1.PostgresPublication,
) error {
	var tableIdentifiers []string
	for _, table := range publication.Tables {
		tableIdentifier := pgx.Identifier{table.Schema, table.Name}
		tableIdentifiers = append(tableIdentifiers, tableIdentifier.Sanitize())
	}

	query := fmt.Sprintf(
		"ALTER PUBLICATION %s SET TABLE %s",
		pgx.Identifier{publication.Name}.Sanitize(),
		strings.Join(tableIdentifiers, ", "),
	)

	r.Log.Info(fmt.Sprintf("executing query: %s", query))
	if _, err := conn.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to set table to publication: %w", err)
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
