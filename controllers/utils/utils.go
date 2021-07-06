package utils

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// contextKey represents a value that will be used as a key in Go's context API.
type contextKey string

const (
	contextKeyRequestLogger contextKey = "request-logger"
)

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
