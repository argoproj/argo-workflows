package auth

import (
	"context"
	"encoding/base64"

	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
)

type ContextKey string

const (
	WfKey   ContextKey = "versioned.Interface"
	KubeKey ContextKey = "kubernetes.Interface"
	restKey ContextKey = "rest.config"
)

const (
	Client = "client"
	Server = "server"
	Hybrid = "hybrid"
)

const V1_Auth_Token = "v1: "

type Gatekeeper struct {
	enableClientAuth string
	// global clients, not to be used if there are better ones
	wfClient   versioned.Interface
	kubeClient kubernetes.Interface
	restConfig *rest.Config
}

func NewGatekeeper(enableClientAuth string, wfClient versioned.Interface, kubeClient kubernetes.Interface, restConfig *rest.Config) Gatekeeper {
	return Gatekeeper{enableClientAuth, wfClient, kubeClient, restConfig}
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
	return s.enableClientAuth == Server
}
func (s Gatekeeper) useHybridAuth() bool {
	return s.enableClientAuth == Hybrid
}

func (s Gatekeeper) useClientAuth(md metadata.MD) (bool, error) {
	if s.enableClientAuth == Client && len(md.Get("grpcgateway-authorization")) == 0 {
		return false, status.Error(codes.Unauthenticated, "Auth Token is not found")
	}
	if s.useHybridAuth() && len(md.Get("grpcgateway-authorization")) > 0 {
		return true, nil
	}
	return true, nil
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
	useClientAuth, err := s.useClientAuth(md)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "Auth Token is not present in the request: %v", err)
	}
	if !useClientAuth {
		return s.wfClient, s.kubeClient, nil
	}

	authorization := md.Get("grpcgateway-authorization")
	token := strings.TrimPrefix(authorization[0], "Bearer ")
	token = strings.TrimPrefix(token, V1_Auth_Token)
	authToken, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "Invalid token found in Authorization header %s: %v", token, err)
	}
	var restConfig *rest.Config
	if useClientAuth {
		restConfig = s.restConfig
		restConfig.BearerToken = string(authToken)
	}

	wfClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "failure to create wfClientset with ClientConfig '%+v': %s", restConfig, err)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "failure to create kubeClientset with ClientConfig '%+v': %s", restConfig, err)
	}
	return wfClient, kubeClient, nil
}
