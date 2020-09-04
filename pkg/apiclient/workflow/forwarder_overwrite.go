package workflow

import (
	"github.com/argoproj/pkg/grpc/http"
)

func init() {
	forward_WorkflowService_WatchWorkflows_0 = http.StreamForwarder
	forward_WorkflowService_WatchEvents_0 = http.StreamForwarder
	forward_WorkflowService_PodLogs_0 = http.StreamForwarder
	forward_WorkflowService_ListWorkflows_0 = http.UnaryForwarder
	forward_WorkflowService_GetWorkflow_0 = http.UnaryForwarder
}
