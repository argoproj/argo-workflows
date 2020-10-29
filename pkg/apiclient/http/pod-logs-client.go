package http

import (
	"encoding/json"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type podLogsClient struct{ clientStream }

func (f *podLogsClient) Recv() (*workflowpkg.LogEntry, error) {
	data, err := f.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	out := &workflowpkg.LogEntry{}
	return out, json.Unmarshal(data, out)
}
