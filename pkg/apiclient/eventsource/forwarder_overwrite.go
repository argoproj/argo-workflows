package eventsource

import (
	"github.com/argoproj/argo-workflows/v4/util/grpc/gateway"
)

func init() {
	forward_EventSourceService_EventSourcesLogs_0 = gateway.StreamForwarder
	forward_EventSourceService_WatchEventSources_0 = gateway.StreamForwarder
}
