package workflow

import (
	"github.com/argoproj/argo/util/http"
)

func init() {
	forward_WorkflowService_WatchWorkflows_0 = http.StreamForwarder
	forward_WorkflowService_PodLogs_0 = http.StreamForwarder
}
