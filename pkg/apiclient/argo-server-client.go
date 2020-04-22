package apiclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	clusterworkflowtmplpkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
)

const (
	// MaxGRPCMessageSize contains max grpc message size supported by the client
	MaxClientGRPCMessageSize = 100 * 1024 * 1024
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

func (a *argoServerClient) NewWorkflowTemplateServiceClient() workflowtemplatepkg.WorkflowTemplateServiceClient {
	return workflowtemplatepkg.NewWorkflowTemplateServiceClient(a.ClientConn)
}

func (a *argoServerClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return workflowarchivepkg.NewArchivedWorkflowServiceClient(a.ClientConn), nil
}

func (a *argoServerClient) NewClusterWorkflowTemplateServiceClient() clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient {
	return clusterworkflowtmplpkg.NewClusterWorkflowTemplateServiceClient(a.ClientConn)
}

func (a *argoServerClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return infopkg.NewInfoServiceClient(a.ClientConn), nil
}

func NewClientConn(argoServer string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(argoServer, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxClientGRPCMessageSize)), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func newContext(auth string) context.Context {
	if auth == "" {
		return context.Background()
	}
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", auth))
}
