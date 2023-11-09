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
		stackTraceBuffer := make([]byte, size)
		stackSize := runtime.Stack(stackTraceBuffer, false)
		// Free up the unused spaces
		stackTraceBuffer = stackTraceBuffer[:stackSize]
		log.Errorf("recovered from panic %q. Call stack:\n%s",
			r,
			stackTraceBuffer)
	}
}
