package event

import (
	"github.com/argoproj/pkg/grpc/http"
)

func init() {
	forward_EventService_WatchWorkflowEventBindings_0 = http.StreamForwarder
}
