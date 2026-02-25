package runtime

import (
	"context"
	"runtime"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// RecoverFromPanic recovers from a panic and logs the panic and call stack
func RecoverFromPanic(ctx context.Context, log logging.Logger) {
	if r := recover(); r != nil {
		// Same as stdlib http server code. Manually allocate stack trace buffer size
		// to prevent excessively large logs
		const size = 64 << 10
		stackTraceBuffer := make([]byte, size)
		stackSize := runtime.Stack(stackTraceBuffer, false)
		// Free up the unused spaces
		stackTraceBuffer = stackTraceBuffer[:stackSize]
		log.WithFields(logging.Fields{
			"error": r,
			"stack": stackTraceBuffer,
		}).Error(ctx, "recovered from panic")
	}
}
