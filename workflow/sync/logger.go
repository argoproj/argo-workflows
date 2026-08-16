package sync

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type loggerFn func(context.Context) logging.Logger

type syncLogger struct {
	name     string
	lockType lockTypeName
}

func (l *syncLogger) get(ctx context.Context) logging.Logger {
	return logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
		"lockType": l.lockType,
		"name":     l.name,
	})
}
