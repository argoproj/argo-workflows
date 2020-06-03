package auth

import (
	"context"
	"fmt"
	"net/http"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/auth/rbac"
	"github.com/argoproj/argo/server/auth/sso"
	"github.com/argoproj/argo/util/kubeconfig"
)

type ContextKey string

const (
	WfKey   ContextKey = "versioned.Interface"
	KubeKey ContextKey = "kubernetes.Interface"
)

type Gatekeeper interface {
	Context(ctx context.Context) (context.Context, error)
	UnaryServerInterceptor() grpc.UnaryServerInterceptor
	StreamServerInterceptor() grpc.StreamServerInterceptor
}

type gatekeeper struct {
	modes     Modes
	namespace string
	// global clients, not to be used if there are better ones
	wfClient   versioned.Interface
	kubeClient kubernetes.Interface
	restConfig *rest.Config
	ssoIf      sso.Interface
	rbacIf     rbac.Interface
}

func NewGatekeeper(modes Modes, namespace string, wfClient versioned.Interface, kubeClient kubernetes.Interface, restConfig *rest.Config, ssoIf sso.Interface, rbacIf rbac.Interface) (Gatekeeper, error) {
	if len(modes) == 0 {
		return nil, fmt.Errorf("must specify at least one auth mode")
	}
	return &gatekeeper{modes, namespace, wfClient, kubeClient, restConfig, ssoIf, rbacIf}, nil
}

func (s *gatekeeper) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx, err = s.Context(ctx)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (s *gatekeeper) StreamServerInterceptor() grpc.StreamServerInterceptor {
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

func (s *gatekeeper) Context(ctx context.Context) (context.Context, error) {
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

func (s gatekeeper) getClients(ctx context.Context) (versioned.Interface, kubernetes.Interface, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	authorization := getAuthHeader(md)
	mode, err := GetMode(authorization)
	if err != nil {
		return nil, nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if !s.modes[mode] {
		return nil, nil, status.Errorf(codes.Unauthenticated, "no valid authentication methods found for mode %v", mode)
	}
	switch mode {
	case Client:
		restConfig, err := kubeconfig.GetRestConfig(authorization)
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
	case Server:
		return s.wfClient, s.kubeClient, nil
	case SSO:
		claims, err := s.ssoIf.Authorize(ctx, authorization)
		if err != nil {
			return nil, nil, status.Error(codes.Unauthenticated, err.Error())
		}
		wfClient, kubeClient, err := s.getClientsFromRBAC(claims)
		if err != nil {
			return nil, nil, err
		}
		return wfClient, kubeClient, nil
	default:
		panic("this should never happen")
	}
}

func (s gatekeeper) getClientsFromRBAC(claims *sso.Claims) (versioned.Interface, kubernetes.Interface, error) {
	serviceAccount, err := s.rbacIf.ServiceAccount(claims.Groups)
	if err != nil {
		return nil, nil, status.Errorf(codes.PermissionDenied, "failed to determine RBAC service account: %v", err.Error())
	}
	account, err := s.kubeClient.CoreV1().ServiceAccounts(s.namespace).Get(serviceAccount.Name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to get service account: %v", err.Error())
	}
	for _, secret := range account.Secrets {
		secret, err := s.kubeClient.CoreV1().Secrets(s.namespace).Get(secret.Name, metav1.GetOptions{})
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "failed to get secret: %v", err.Error())
		}
		token, ok := secret.Data["token"]
		if ok {
			restConfig, err := kubeconfig.GetBearerRestConfig(string(token))
			if err != nil {
				return nil, nil, status.Errorf(codes.Internal, "failed to create REST config: %v", err)
			}
			wfClient, err := versioned.NewForConfig(restConfig)
			if err != nil {
				return nil, nil, status.Errorf(codes.Internal, "failed to create workflow client: %v", err)
			}
			kubeClient, err := kubernetes.NewForConfig(restConfig)
			if err != nil {
				return nil, nil, status.Errorf(codes.Internal, "failed to create kube client: %v", err)
			}
			return wfClient, kubeClient, nil
		}
	}
	return nil, nil, status.Errorf(codes.Internal, `could not find secret for service account named "%s"`, account.Name)
}
