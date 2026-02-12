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
	syncpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sync"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	grpcutil "github.com/argoproj/argo-workflows/v3/util/grpc"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

const (
	// MaxGRPCMessageSize contains max grpc message size supported by the client
	MaxClientGRPCMessageSize = 100 * 1024 * 1024
)

type argoServerClient struct {
	*grpc.ClientConn
}

var _ Client = &argoServerClient{}

func newArgoServerClient(ctx context.Context, opts ArgoServerOpts, auth string) (context.Context, Client, error) {
	conn, err := newClientConn(opts)
	if err != nil {
		return nil, nil, err
	}
	return newContext(ctx, auth), &argoServerClient{conn}, nil
}

func (a *argoServerClient) NewWorkflowServiceClient(_ context.Context) workflowpkg.WorkflowServiceClient {
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

func (a *argoServerClient) NewSyncServiceClient(_ context.Context) (syncpkg.SyncServiceClient, error) {
	return syncpkg.NewSyncServiceClient(a.ClientConn), nil
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

func newContext(ctx context.Context, auth string) context.Context {

	bgCtx := logging.RequireLoggerFromContext(ctx).NewBackgroundContext()
	if auth == "" {
		return ctx
	}
	return metadata.NewOutgoingContext(bgCtx, metadata.Pairs("authorization", auth))
}
