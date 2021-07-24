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
	"sigs.k8s.io/controller-runtime/pkg/log"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
	"github.com/glints-dev/postgres-config-operator/controllers/utils"
	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v4"
)

// PostgresGrantReconciler reconciles a PostgresGrant object
type PostgresGrantReconciler struct {
	client.Client
	recorder record.EventRecorder
	Log      logr.Logger
	Scheme   *runtime.Scheme
}

//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgresgrants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgresgrants/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=postgres.glints.com,resources=postgresgrants/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PostgresGrant object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *PostgresGrantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	ctx = utils.WithRequestLogger(ctx, req, "postgresgrant", r.Log)

	grant := &postgresv1alpha1.PostgresGrant{}
	if err := r.Get(ctx, req.NamespacedName, grant); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	conn, err := utils.SetupPostgresConnection(
		ctx,
		r,
		r.recorder,
		grant.Spec.PostgresRef,
		grant.ObjectMeta,
	)
	if err != nil {
		r.recorder.Eventf(
			grant,
			corev1.EventTypeWarning,
			EventTypeFailedSetupPostgresConnection,
			err.Error(),
		)
		return ctrl.Result{Requeue: true}, nil
	}
	defer conn.Close(ctx)

	objectTypesToReconcile := []postgresv1alpha1.ObjectType{
		postgresv1alpha1.ObjectTypeDatabase,
		postgresv1alpha1.ObjectTypeFunction,
		postgresv1alpha1.ObjectTypeSchema,
		postgresv1alpha1.ObjectTypeSequence,
		postgresv1alpha1.ObjectTypeTable,
	}

	for _, objectType := range objectTypesToReconcile {
		reconcileResult := r.reconcileObjectGrants(
			ctx,
			conn,
			grant,
			objectType,
		)
		if reconcileResult.Requeue {
			return reconcileResult, nil
		}
	}

	r.recorder.Event(
		grant,
		corev1.EventTypeNormal,
		EventTypeSuccessfulReconcile,
		"successfully reconcilled grant",
	)

	return ctrl.Result{}, nil
}

func (r *PostgresGrantReconciler) reconcileObjectGrants(
	ctx context.Context,
	conn *pgx.Conn,
	grant *postgresv1alpha1.PostgresGrant,
	objectType postgresv1alpha1.ObjectType,
) ctrl.Result {
	privileges := grant.PrivilegesForType(objectType)
	if len(privileges) == 0 {
		privileges = []string{"ALL"}
	}

	for _, objectIdentifier := range grant.IdentifiersForType(objectType) {
		roleIdentifier := pgx.Identifier{grant.Spec.Role}

		sql := fmt.Sprintf(
			`GRANT %s ON %s TO %s`,
			strings.Join(privileges, ", "), // Already validated by admission webhook
			objectIdentifier.Sanitize(),
			roleIdentifier.Sanitize(),
		)

		if _, err := conn.Exec(ctx, sql); err != nil {
			r.recorder.Eventf(
				grant,
				corev1.EventTypeWarning,
				EventTypeFailedReconcile,
				"failed to reconcile %s grant: %v",
				objectType,
				err,
			)
			return ctrl.Result{Requeue: true}
		}
	}

	return ctrl.Result{}
}

// SetupWithManager sets up the controller with the Manager.
func (r *PostgresGrantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("postgres-grant-controller")

	return ctrl.NewControllerManagedBy(mgr).
		For(&postgresv1alpha1.PostgresGrant{}).
		Complete(r)
}
