package pipeline

import (
	"github.com/argoproj/pkg/grpc/http"
)

func init() {
	forward_PipelineService_WatchPipelines_0 = http.StreamForwarder
	forward_PipelineService_PipelineLogs_0 = http.StreamForwarder
	forward_PipelineService_WatchSteps_0 = http.StreamForwarder
}
