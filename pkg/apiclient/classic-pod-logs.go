package apiclient

import (
	"context"

	"google.golang.org/grpc/metadata"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type classicPodLogs struct {
	logEntries chan *workflowpkg.LogEntry
}

func newClassicPodLogs() *classicPodLogs {
	return &classicPodLogs{make(chan *workflowpkg.LogEntry, 515)}
}

func (c *classicPodLogs) Send(logEntry *workflowpkg.LogEntry) error {
	c.logEntries <- logEntry
	return nil
}

func (c *classicPodLogs) SetHeader(metadata.MD) error {
	panic("implement me")
}

func (c *classicPodLogs) SendHeader(metadata.MD) error {
	panic("implement me")
}

func (c *classicPodLogs) SetTrailer(metadata.MD) {
	panic("implement me")
}

func (c *classicPodLogs) Recv() (*workflowpkg.LogEntry, error) {
	logEntry := <-c.logEntries
	return logEntry, nil
}

func (c *classicPodLogs) Header() (metadata.MD, error) {
	panic("implement me")
}

func (c *classicPodLogs) Trailer() metadata.MD {
	panic("implement me")
}

func (c *classicPodLogs) CloseSend() error {
	panic("implement me")
}

func (c *classicPodLogs) Context() context.Context {
	panic("implement me")
}

func (c *classicPodLogs) SendMsg(interface{}) error {
	panic("implement me")
}

func (c *classicPodLogs) RecvMsg(interface{}) error {
	panic("implement me")
}
