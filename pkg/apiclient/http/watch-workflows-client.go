package http

import (
	"encoding/json"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

const prefixLength = len("data: ")

type watchWorkflowsClient struct{ clientStream }

func (f watchWorkflowsClient) Recv() (*workflowpkg.WorkflowWatchEvent, error) {
	for {
		data, err := f.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		if len(data) <= prefixLength {
			continue
		}
		out := &workflowpkg.WorkflowWatchEvent{}
		return out, json.Unmarshal(data[prefixLength:], out)
	}
}
