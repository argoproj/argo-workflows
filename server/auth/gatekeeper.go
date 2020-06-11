package auth

import (
	"context"
	"net/http"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/util/kubeconfig"
)

type ContextKey string

const (
	WfKey   ContextKey = "versioned.Interface"
	KubeKey ContextKey = "kubernetes.Interface"
)

const (
	Client = "client"
	Server = "server"
	Hybrid = "hybrid"
)

type Gatekeeper struct {
	authType string
	// global clients, not to be used if there are better ones
	wfClient   versioned.Interface
	kubeClient kubernetes.Interface
	restConfig *rest.Config
}

func NewGatekeeper(authType string, wfClient versioned.Interface, kubeClient kubernetes.Interface, restConfig *rest.Config) Gatekeeper {
	return Gatekeeper{authType, wfClient, kubeClient, restConfig}
}

func (s *Gatekeeper) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx, err = s.Context(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (s *Gatekeeper) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := s.Context(ss.Context())
		if err != nil {
			return err
		}
		wrapped := grpc_middleware.WrapServerStream(ss)
		wrapped.WrappedContext = ctx
		return handler(srv, wrapped)
	}
}

func (s *Gatekeeper) Context(ctx context.Context) (context.Context, error) {
	wfClient, kubeClient, err := s.getClients(ctx)
	if err != nil {
		return nil, err
	}
	return context.WithValue(context.WithValue(ctx, WfKey, wfClient), KubeKey, kubeClient), nil
}

func GetWfClient(ctx context.Context) versioned.Interface {
	return ctx.Value(WfKey).(versioned.Interface)
}

func GetKubeClient(ctx context.Context) kubernetes.Interface {
	return ctx.Value(KubeKey).(kubernetes.Interface)
}
func (s Gatekeeper) useServerAuth() bool {
	return s.authType == Server
}
func (s Gatekeeper) useHybridAuth() bool {
	return s.authType == Hybrid
}

func (s Gatekeeper) useClientAuth(token string) bool {
	if s.authType == Client {
		return true
	}
	if s.useHybridAuth() && token != "" {
		return true
	}
	return false
}

func getAuthHeader(md metadata.MD) string {
	// looks for the HTTP header `Authorization: Bearer ...`
	for _, t := range md.Get("authorization") {
		return t
	}
	// check the HTTP cookie
	for _, t := range md.Get("grpcgateway-cookie") {
		header := http.Header{}
		header.Add("Cookie", t)
		request := http.Request{Header: header}
		token, err := request.Cookie("authorization")
		if err == nil {
			return token.Value
		}
	}
	return ""
}

func (s Gatekeeper) getClients(ctx context.Context) (versioned.Interface, kubernetes.Interface, error) {

	if s.useServerAuth() {
		return s.wfClient, s.kubeClient, nil
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		if s.useHybridAuth() {
			return s.wfClient, s.kubeClient, nil
		}
		return nil, nil, status.Error(codes.Unauthenticated, "unable to get metadata from incoming context")
	}

	authString := getAuthHeader(md)

	if !s.useClientAuth(authString) {
		return s.wfClient, s.kubeClient, nil
	}

	restConfig, err := kubeconfig.GetRestConfig(authString)

	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "failed to create REST config: %v", err)
	}

	wfClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "failure to create wfClientset with ClientConfig: %v", err)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "failure to create kubeClientset with ClientConfig: %v", err)
	}
	return wfClient, kubeClient, nil
}
