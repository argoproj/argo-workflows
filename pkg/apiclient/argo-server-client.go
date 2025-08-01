package apiclient

import (
	"context"
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	clusterworkflowtmplpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	grpcutil "github.com/argoproj/argo-workflows/v3/util/grpc"
)

const (
	// MaxGRPCMessageSize contains max grpc message size supported by the client
	MaxClientGRPCMessageSize = 100 * 1024 * 1024
)

type argoServerClient struct {
	*grpc.ClientConn
}

var _ Client = &argoServerClient{}

func newArgoServerClient(opts ArgoServerOpts, auth string) (context.Context, Client, error) {
	conn, err := newClientConn(opts)
	if err != nil {
		return nil, nil, err
	}
	return newContext(auth), &argoServerClient{conn}, nil
}

func (a *argoServerClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return workflowpkg.NewWorkflowServiceClient(a.ClientConn)
}

func (a *argoServerClient) NewCronWorkflowServiceClient() (cronworkflowpkg.CronWorkflowServiceClient, error) {
	return cronworkflowpkg.NewCronWorkflowServiceClient(a.ClientConn), nil
}

func (a *argoServerClient) NewWorkflowTemplateServiceClient() (workflowtemplatepkg.WorkflowTemplateServiceClient, error) {
	return workflowtemplatepkg.NewWorkflowTemplateServiceClient(a.ClientConn), nil
}

func (a *argoServerClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return workflowarchivepkg.NewArchivedWorkflowServiceClient(a.ClientConn), nil
}

func (a *argoServerClient) NewClusterWorkflowTemplateServiceClient() (clusterworkflowtmplpkg.ClusterWorkflowTemplateServiceClient, error) {
	return clusterworkflowtmplpkg.NewClusterWorkflowTemplateServiceClient(a.ClientConn), nil
}

func (a *argoServerClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return infopkg.NewInfoServiceClient(a.ClientConn), nil
}

func newClientConn(opts ArgoServerOpts) (*grpc.ClientConn, error) {
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	if opts.Secure {
		creds = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: opts.InsecureSkipVerify}))
	}
	conn, err := grpc.NewClient(opts.URL,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxClientGRPCMessageSize)),
		creds,
		grpc.WithUnaryInterceptor(grpcutil.GetVersionHeaderClientUnaryInterceptor),
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
