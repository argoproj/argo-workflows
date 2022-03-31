package eventsource

import (
	"github.com/argoproj/pkg/grpc/http"
)

func init() {
	forward_EventSourceService_WatchEventSourcesLogs_0 = http.StreamForwarder
	forward_EventSourceService_WatchEventSources_0 = http.StreamForwarder
}
