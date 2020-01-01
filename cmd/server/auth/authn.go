package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
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
)

type AuthN struct {
	enableClientAuth bool
	// global clients, not to be used if there are better ones
	wfClient   versioned.Interface
	kubeClient kubernetes.Interface
}

func NewAuthN(enableClientAuth bool, wfClient versioned.Interface, kubeClient kubernetes.Interface) AuthN {
	return AuthN{enableClientAuth, wfClient, kubeClient}
}

func (s AuthN) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		wfClient, kubeClient, err := s.getClients(ctx)
		if err != nil {
			return nil, err
		}
		return handler(context.WithValue(context.WithValue(ctx, WfKey, wfClient), KubeKey, kubeClient), req)
	}
}

func (s AuthN) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wfClient, kubeClient, err := s.getClients(ss.Context())
		if err != nil {
			return err
		}
		wrapped := grpc_middleware.WrapServerStream(ss)
		wrapped.WrappedContext = context.WithValue(context.WithValue(ss.Context(), WfKey, wfClient), KubeKey, kubeClient)
		return handler(srv, wrapped)
	}
}

func GetWfClient(ctx context.Context) versioned.Interface {
	return ctx.Value(WfKey).(versioned.Interface)
}

func GetKubeClient(ctx context.Context) kubernetes.Interface {
	return ctx.Value(KubeKey).(kubernetes.Interface)
}

func (s AuthN) getClients(ctx context.Context) (versioned.Interface, kubernetes.Interface, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		if !s.enableClientAuth {
			return s.wfClient, s.kubeClient, nil
		}
		return nil, nil, fmt.Errorf("unable to get metadata from incoming context")
	}
	authorization := md.Get("grpcgateway-authorization")
	if len(authorization) == 0 {
		if !s.enableClientAuth {
			return s.wfClient, s.kubeClient, nil
		}
		return nil, nil, status.Error(codes.Unauthenticated, "Authorization header not found")
	}
	token := strings.TrimPrefix(authorization[0], "Bearer ")
	restConfigBytes, err := base64.StdEncoding.DecodeString(token)

	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "Invalid token found in Authorization header %s: %v", token, err)
	}

	var restConfig rest.Config
	err = json.Unmarshal(restConfigBytes, &restConfig)
	if err != nil {
		return nil, nil, err
	}

	if s.enableClientAuth {
		// we want to prevent people using in-cluster set-up
		if restConfig.BearerTokenFile != "" || restConfig.CAFile != "" || restConfig.CertFile != "" || restConfig.KeyFile != "" {
			return nil, nil, status.Errorf(codes.Unauthenticated, "illegal bearer token")
		}
		host := strings.SplitN(restConfig.Host, ":", 2)[0]
		if host == "localhost" || net.ParseIP(host).IsLoopback() {
			return nil, nil, status.Errorf(codes.Unauthenticated, "illegal bearer token")
		}
	}

	wfClient, err := versioned.NewForConfig(&restConfig)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "failure to create wfClientset with ClientConfig '%+v': %s", restConfig, err)
	}
	kubeClient, err := kubernetes.NewForConfig(&restConfig)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "failure to create kubeClientset with ClientConfig '%+v': %s", restConfig, err)
	}
	return wfClient, kubeClient, nil
}
