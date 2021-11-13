package utils

import (
	"context"
	"fmt"
	"net/url"

	"github.com/go-logr/logr"
	"github.com/jackc/pgx/v4"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	postgresv1alpha1 "github.com/glints-dev/postgres-config-operator/api/v1alpha1"
)

// contextKey represents a value that will be used as a key in Go's context API.
type contextKey string

const (
	contextKeyRequestLogger contextKey = "request-logger"
)

func SetupPostgresConnection(
	ctx context.Context,
	r client.Reader,
	recorder record.EventRecorder,
	ref postgresv1alpha1.PostgresRef,
	obj metav1.ObjectMeta,
) (*pgx.Conn, error) {
	secretNamespacedName := types.NamespacedName{
		Name:      ref.SecretRef.SecretName,
		Namespace: obj.GetNamespace(),
	}
	secretRef := &corev1.Secret{}
	if err := r.Get(ctx, secretNamespacedName, secretRef); err != nil {
		return nil, &ErrGetSecret{Err: err}
	}

	connURL := url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			string(secretRef.Data["POSTGRES_USER"]),
			string(secretRef.Data["POSTGRES_PASSWORD"]),
		),
		Host:    fmt.Sprintf("%s:%d", ref.Host, ref.Port),
		Path:    fmt.Sprintf("/%s", ref.Database),
		RawPath: ref.Database,
	}
	conn, err := pgx.Connect(ctx, connURL.String())
	if err != nil {
		return nil, &ErrFailedConnectPostgres{Err: err}
	}

	return conn, nil
}

// WithRequestLogger returns a context that has a request-scoped logger attached
// to it.
func WithRequestLogger(
	ctx context.Context,
	req ctrl.Request,
	resourceKind string,
	baseLogger logr.Logger,
) context.Context {
	logger := baseLogger.WithValues(resourceKind, req.NamespacedName)
	return context.WithValue(ctx, contextKeyRequestLogger, logger)
}

// WithRequestLogger returns request-scoped logger from the given context.
func RequestLogger(ctx context.Context, baseLogger logr.Logger) logr.Logger {
	if logger, ok := ctx.Value(contextKeyRequestLogger).(logr.Logger); ok {
		return logger
	} else {
		return baseLogger
	}
}
