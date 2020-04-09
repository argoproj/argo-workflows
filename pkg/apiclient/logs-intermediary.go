package apiclient

import (
	"context"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type logsIntermediary struct {
	abstractIntermediary
	logEntries chan *workflowpkg.LogEntry
}

func (c *logsIntermediary) Send(logEntry *workflowpkg.LogEntry) error {
	c.logEntries <- logEntry
	return nil
}

func (c *logsIntermediary) Recv() (*workflowpkg.LogEntry, error) {
	select {
	case err := <-c.error:
		return nil, err
	case logEntry := <-c.logEntries:
		return logEntry, nil
	}
}

func newLogsIntermediary(ctx context.Context) *logsIntermediary {
	return &logsIntermediary{newAbstractIntermediary(ctx), make(chan *workflowpkg.LogEntry)}
}
