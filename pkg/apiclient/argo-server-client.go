package apiclient

import (
	"context"
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	clusterworkflowtmplpkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
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

func newArgoServerClient(argoServer, auth string, secure, insecureSkipVerify bool) (context.Context, Client, error) {
	conn, err := newClientConn(argoServer, secure, insecureSkipVerify)
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

func newClientConn(argoServer string, secure, insecureSkipVerify bool) (*grpc.ClientConn, error) {
	creds := grpc.WithInsecure()
	if secure {
		creds = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: insecureSkipVerify}))
	}
	conn, err := grpc.Dial(argoServer,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxClientGRPCMessageSize)),
		creds,
	)
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
