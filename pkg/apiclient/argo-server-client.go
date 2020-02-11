package apiclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
)

type argoServerClient struct {
	*grpc.ClientConn
}

func newArgoServerClient(argoServer, auth string) (context.Context, Client, error) {
	conn, err := NewClientConn(argoServer)
	if err != nil {
		return nil, nil, err
	}
	return newContext(auth), &argoServerClient{conn}, nil
}

func (a *argoServerClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return workflowpkg.NewWorkflowServiceClient(a.ClientConn)
}

func (a *argoServerClient) NewCronWorkflowServiceClient() cronworkflowpkg.CronWorkflowServiceClient {
	return cronworkflowpkg.NewCronWorkflowServiceClient(a.ClientConn)
}

func (a *argoServerClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return workflowarchivepkg.NewArchivedWorkflowServiceClient(a.ClientConn), nil
}

func NewClientConn(argoServer string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(argoServer, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// DEPRECATED
func NewContext(auth string) context.Context {
	return newContext(auth)
}

func newContext(auth string) context.Context {
	if auth == "" {
		return context.Background()
	}
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", auth))
}
