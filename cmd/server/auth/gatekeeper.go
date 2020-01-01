package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"strings"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	authorizationv1 "k8s.io/api/authorization/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
)

type ContextKey string

const (
	WfKey   ContextKey = "versioned.Interface"
	KubeKey ContextKey = "kubernetes.Interface"
)

type Gatekeeper struct {
	enableClientAuth bool
	// global clients, not to be used if there are better ones
	wfClient   versioned.Interface
	kubeClient kubernetes.Interface
}

func NewGatekeeper(enableClientAuth bool, wfClient versioned.Interface, kubeClient kubernetes.Interface) Gatekeeper {
	return Gatekeeper{enableClientAuth, wfClient, kubeClient}
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

func CanI(ctx context.Context, verb, resource, namespace, name string) (bool, error) {
	kubeClientset := GetKubeClient(ctx)
	review, err := kubeClientset.AuthorizationV1().SelfSubjectAccessReviews().Create(&authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      verb,
				Group:     "argoproj.io",
				Resource:  resource,
				Name:      name,
			},
		},
	})
	if err != nil {
		return false, err
	}
	return review.Status.Allowed, nil
}

func (s Gatekeeper) getClients(ctx context.Context) (versioned.Interface, kubernetes.Interface, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		if !s.enableClientAuth {
			return s.wfClient, s.kubeClient, nil
		}
		return nil, nil, status.Error(codes.Unauthenticated, "unable to get metadata from incoming context")
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
