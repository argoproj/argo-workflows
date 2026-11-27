package workflow

import (
	"github.com/argoproj/argo-workflows/v4/util/grpc/gateway"
)

func init() {
	forward_WorkflowService_WatchWorkflows_0 = gateway.StreamForwarder
	forward_WorkflowService_WatchEvents_0 = gateway.StreamForwarder
	forward_WorkflowService_PodLogs_0 = gateway.StreamForwarder
	forward_WorkflowService_WorkflowLogs_0 = gateway.StreamForwarder
}
