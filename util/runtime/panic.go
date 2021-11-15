package runtime

import (
	"runtime"

	log "github.com/sirupsen/logrus"
)

func RecoverFromPanic(log *log.Entry) {
	if r := recover(); r != nil {
		// Same as stdlib http server code. Manually allocate stack trace buffer size
		// to prevent excessively large logs
		const size = 64 << 10
		stacktrace := make([]byte, size)
		stacktrace = stacktrace[:runtime.Stack(stacktrace, false)]
		log.Errorf("recovered from panic %q. Call stack:\n%s",
			r,
			stacktrace)
	}
}
