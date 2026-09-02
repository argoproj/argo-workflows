package http1

import (
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
)

type eventWatchClient struct{ serverSentEventsClient }

func (f eventWatchClient) Recv() (*workflowpkg.EventWatchEvent, error) {
	v := &workflowpkg.EventWatchEvent{}
	return v, f.RecvEvent(v)
}
