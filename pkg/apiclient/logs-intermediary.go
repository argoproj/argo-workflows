package apiclient

import (
	"context"

	"google.golang.org/grpc/metadata"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
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

func (c *logsIntermediary) SendHeader(metadata.MD) error {
	// We invoke `SendHeader` in order to eagerly flush headers to allow us to send period
	// keepalives when using HTTP/1 and Server Sent Events, so we need to implement this here,
	// though we don't use the meta for anything.
	return nil
}

func newLogsIntermediary(ctx context.Context) *logsIntermediary {
	return &logsIntermediary{newAbstractIntermediary(ctx), make(chan *workflowpkg.LogEntry)}
}
