package workflow

import (
	"github.com/argoproj/pkg/grpc/http"
)

func init() {
	forward_WorkflowService_WatchWorkflows_0 = http.StreamForwarder
	forward_WorkflowService_WatchEvents_0 = http.StreamForwarder
	forward_WorkflowService_WatchPodLogs_0 = http.StreamForwarder
	forward_WorkflowService_WatchWorkflowLogs_0 = http.StreamForwarder
}
