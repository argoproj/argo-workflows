package apiclient

import (
	"context"
	"io"

	"google.golang.org/grpc/metadata"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

// The "Poison pill pattern" to tell the channel to close.
var closeTheChan *workflowpkg.LogEntry

type logsIntermediary struct {
	logEntries chan *workflowpkg.LogEntry
}

func (c *logsIntermediary) Send(logEntry *workflowpkg.LogEntry) error {
	c.logEntries <- logEntry
	return nil
}

func (c *logsIntermediary) Recv() (*workflowpkg.LogEntry, error) {
	logEntry := <-c.logEntries
	if logEntry == closeTheChan {
		return nil, io.EOF
	}
	return logEntry, nil
}

func newLogsIntermediary() *logsIntermediary {
	return &logsIntermediary{make(chan *workflowpkg.LogEntry, 512)}
}

func (c *logsIntermediary) SetHeader(metadata.MD) error {
	panic("implement me")
}

func (c *logsIntermediary) SendHeader(metadata.MD) error {
	panic("implement me")
}

func (c *logsIntermediary) SetTrailer(metadata.MD) {
	panic("implement me")
}

func (c *logsIntermediary) Header() (metadata.MD, error) {
	panic("implement me")
}

func (c *logsIntermediary) Trailer() metadata.MD {
	panic("implement me")
}

func (c *logsIntermediary) CloseSend() error {
	c.logEntries <- closeTheChan
	return nil
}

func (c *logsIntermediary) Context() context.Context {
	panic("implement me")
}

func (c *logsIntermediary) SendMsg(interface{}) error {
	panic("implement me")
}

func (c *logsIntermediary) RecvMsg(interface{}) error {
	panic("implement me")
}
