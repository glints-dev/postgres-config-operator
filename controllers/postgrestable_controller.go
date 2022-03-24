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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
	"github.com/glints-dev/postgres-config-operator/controllers/utils"
	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v4"
)

// PostgresTableReconciler reconciles a PostgresTable object
type PostgresTableReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
	Log      logr.Logger
}

//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgrestables,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgrestables/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgrestables/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *PostgresTableReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = utils.WithRequestLogger(ctx, req, "postgrestable", r.Log)

	table := &postgresv1alpha1.PostgresTable{}
	if err := r.Get(ctx, req.NamespacedName, table); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	conn, err := utils.SetupPostgresConnection(
		ctx,
		r,
		r.recorder,
		table.Spec.PostgresRef,
		table.ObjectMeta,
	)
	if err != nil {
		r.recorder.Eventf(
			table,
			corev1.EventTypeWarning,
			EventTypeFailedSetupPostgresConnection,
			err.Error(),
		)
		return ctrl.Result{Requeue: true}, nil
	}
	defer conn.Close(ctx)

	r.recorder.Event(
		table,
		corev1.EventTypeNormal,
		EventTypeSuccessConnectPostgres,
		"successfully connected to PostgreSQL",
	)

	reconcileResult, err := r.reconcile(ctx, conn, table)
	if err != nil {
		return reconcileResult, fmt.Errorf("failed to reconcile: %w", err)
	}

	if reconcileResult.Requeue {
		return reconcileResult, nil
	}

	r.recorder.Event(
		table,
		corev1.EventTypeNormal,
		EventTypeSuccessfulReconcile,
		"successfully reconcilled publication",
	)

	return ctrl.Result{}, nil
}

func (r *PostgresTableReconciler) reconcile(
	ctx context.Context,
	conn *pgx.Conn,
	table *postgresv1alpha1.PostgresTable,
) (ctrl.Result, error) {
	r.Log.Info("reconcilling table", "table.Name", table.Name)

	err := r.createTable(ctx, conn, table)
	if err != nil {
		r.recorder.Eventf(
			table,
			corev1.EventTypeWarning,
			EventTypeFailedReconcile,
			"failed to create table: %v",
			err,
		)
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func (r *PostgresTableReconciler) createTable(
	ctx context.Context,
	conn *pgx.Conn,
	table *postgresv1alpha1.PostgresTable,
) error {
	if err := r.ensureSchemaExists(ctx, conn, table); err != nil {
		return fmt.Errorf("failed to ensure schema exists: %w", err)
	}

	sql := r.buildCreateTableQuery(table)
	if _, err := conn.Exec(ctx, sql); err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (r *PostgresTableReconciler) ensureSchemaExists(
	ctx context.Context,
	conn *pgx.Conn,
	table *postgresv1alpha1.PostgresTable,
) error {
	schema := table.Spec.Schema
	sql := fmt.Sprintf(
		"CREATE SCHEMA IF NOT EXISTS %s",
		pgx.Identifier{schema}.Sanitize(),
	)

	if _, err := conn.Exec(ctx, sql); err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (r *PostgresTableReconciler) buildCreateTableQuery(
	table *postgresv1alpha1.PostgresTable,
) string {
	tableIdentifier := pgx.Identifier{
		table.Spec.Schema,
		table.Spec.Name,
	}

	var primaryKeys []string
	for _, column := range table.Spec.Columns {
		if column.PrimaryKey {
			primaryKeys = append(
				primaryKeys,
				pgx.Identifier{column.Name}.Sanitize(),
			)
		}
	}

	var columnClauses []string
	for _, column := range table.Spec.Columns {
		sqlParts := []string{
			pgx.Identifier{column.Name}.Sanitize(),
			pgx.Identifier{column.DataType}.Sanitize(),
		}

		if !column.Nullable {
			sqlParts = append(sqlParts, "NOT NULL")
		}

		columnClauses = append(columnClauses, strings.Join(sqlParts, " "))
	}

	if len(primaryKeys) > 0 {
		columnClauses = append(
			columnClauses,
			fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(primaryKeys, ", ")))
	}

	sql := fmt.Sprintf(
		`CREATE TABLE %s (%s)`,
		tableIdentifier.Sanitize(),
		strings.Join(columnClauses, ",\n"),
	)

	return sql
}

// SetupWithManager sets up the controller with the Manager.
func (r *PostgresTableReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("postgres-table-controller")

	return ctrl.NewControllerManagedBy(mgr).
		For(&postgresv1alpha1.PostgresTable{}).
		Complete(r)
}
