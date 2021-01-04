package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/antonmedv/expr"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/auth/serviceaccount"
	"github.com/argoproj/argo/server/auth/sso"
	"github.com/argoproj/argo/server/auth/types"
	"github.com/argoproj/argo/util/kubeconfig"
	"github.com/argoproj/argo/workflow/common"
)

type ContextKey string

const (
	WfKey     ContextKey = "versioned.Interface"
	KubeKey   ContextKey = "kubernetes.Interface"
	ClaimsKey ContextKey = "types.Claims"
)

//go:generate mockery -name Gatekeeper

type Gatekeeper interface {
	Context(ctx context.Context) (context.Context, error)
	UnaryServerInterceptor() grpc.UnaryServerInterceptor
	StreamServerInterceptor() grpc.StreamServerInterceptor
}

type ClientForAuthorization func(authorization string) (*rest.Config, versioned.Interface, kubernetes.Interface, error)

type gatekeeper struct {
	Modes Modes
	// global clients, not to be used if there are better ones
	wfClient               versioned.Interface
	kubeClient             kubernetes.Interface
	restConfig             *rest.Config
	ssoIf                  sso.Interface
	clientForAuthorization ClientForAuthorization
	// The namespace the server is installed in.
	namespace string
}

func NewGatekeeper(modes Modes, wfClient versioned.Interface, kubeClient kubernetes.Interface, restConfig *rest.Config, ssoIf sso.Interface, clientForAuthorization ClientForAuthorization, namespace string) (Gatekeeper, error) {
	if len(modes) == 0 {
		return nil, fmt.Errorf("must specify at least one auth mode")
	}
	return &gatekeeper{modes, wfClient, kubeClient, restConfig, ssoIf, clientForAuthorization, namespace}, nil
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
	wfClient, kubeClient, claims, err := s.getClients(ctx)
	if err != nil {
		return nil, err
	}
	return context.WithValue(context.WithValue(context.WithValue(ctx, WfKey, wfClient), KubeKey, kubeClient), ClaimsKey, claims), nil
}

func GetWfClient(ctx context.Context) versioned.Interface {
	return ctx.Value(WfKey).(versioned.Interface)
}

func GetKubeClient(ctx context.Context) kubernetes.Interface {
	return ctx.Value(KubeKey).(kubernetes.Interface)
}

func GetClaims(ctx context.Context) *types.Claims {
	config, _ := ctx.Value(ClaimsKey).(*types.Claims)
	return config
}

func getAuthHeader(md metadata.MD) string {
	// looks for the HTTP header `Authorization: Bearer ...`
	for _, t := range md.Get("authorization") {
		return t
	}
	// check the HTTP cookie
	for _, t := range md.Get("cookie") {
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

func (s gatekeeper) getClients(ctx context.Context) (versioned.Interface, kubernetes.Interface, *types.Claims, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	authorization := getAuthHeader(md)
	mode, valid := s.Modes.GetMode(authorization)
	if !valid {
		return nil, nil, nil, status.Error(codes.Unauthenticated, "token not valid for running mode")
	}
	switch mode {
	case Client:
		restConfig, wfClient, kubeClient, err := s.clientForAuthorization(authorization)
		if err != nil {
			return nil, nil, nil, status.Error(codes.Unauthenticated, err.Error())
		}
		claims, _ := serviceaccount.ClaimSetFor(restConfig)
		return wfClient, kubeClient, claims, nil
	case Server:
		claims, _ := serviceaccount.ClaimSetFor(s.restConfig)
		return s.wfClient, s.kubeClient, claims, nil
	case SSO:
		claims, err := s.ssoIf.Authorize(authorization)
		if err != nil {
			return nil, nil, nil, status.Error(codes.Unauthenticated, err.Error())
		}
		if s.ssoIf.IsRBACEnabled() {
			v, k, err := s.rbacAuthorization(claims)
			if err != nil {
				log.WithError(err).Error("failed to perform RBAC authorization")
				return nil, nil, nil, status.Error(codes.PermissionDenied, "not allowed")
			}
			return v, k, claims, nil
		} else {
			return s.wfClient, s.kubeClient, claims, nil
		}
	default:
		panic("this should never happen")
	}
}

func (s *gatekeeper) rbacAuthorization(claims *types.Claims) (versioned.Interface, kubernetes.Interface, error) {
	list, err := s.kubeClient.CoreV1().ServiceAccounts(s.namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list SSO RBAC service accounts: %w", err)
	}
	var serviceAccounts []corev1.ServiceAccount
	for _, serviceAccount := range list.Items {
		_, ok := serviceAccount.Annotations[common.AnnotationKeyRBACRule]
		if !ok {
			continue
		}
		serviceAccounts = append(serviceAccounts, serviceAccount)
	}
	precedence := func(serviceAccount corev1.ServiceAccount) int {
		i, _ := strconv.Atoi(serviceAccount.Annotations[common.AnnotationKeyRBACRulePrecedence])
		return i
	}
	sort.Slice(serviceAccounts, func(i, j int) bool { return precedence(serviceAccounts[i]) > precedence(serviceAccounts[j]) })
	for _, serviceAccount := range serviceAccounts {
		rule := serviceAccount.Annotations[common.AnnotationKeyRBACRule]
		data, err := json.Marshal(claims)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshall claims: %w", err)
		}
		v := make(map[string]interface{})
		err = json.Unmarshal(data, &v)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshall claims: %w", err)
		}
		result, err := expr.Eval(rule, v)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to evaluate rule: %w", err)
		}
		allow, ok := result.(bool)
		if !ok {
			return nil, nil, fmt.Errorf("failed to evaluate rule: not a boolean")
		}
		if !allow {
			continue
		}
		authorization, err := s.authorizationForServiceAccount(serviceAccount.Name)
		if err != nil {
			return nil, nil, err
		}
		_, wfClient, kubeClient, err := s.clientForAuthorization(authorization)
		if err != nil {
			return nil, nil, err
		}
		log.WithFields(log.Fields{"serviceAccount": serviceAccount.Name, "subject": claims.Subject}).Info("selected SSO RBAC service account for user")
		return wfClient, kubeClient, nil
	}
	return nil, nil, fmt.Errorf("no service account rule matches")
}

func (s *gatekeeper) authorizationForServiceAccount(serviceAccountName string) (string, error) {
	serviceAccount, err := s.kubeClient.CoreV1().ServiceAccounts(s.namespace).Get(serviceAccountName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service account: %w", err)
	}
	if len(serviceAccount.Secrets) == 0 {
		return "", fmt.Errorf("expected at least one secret for SSO RBAC service account: %w", err)
	}
	secret, err := s.kubeClient.CoreV1().Secrets(s.namespace).Get(serviceAccount.Secrets[0].Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service account secret: %w", err)
	}
	return "Bearer " + string(secret.Data["token"]), nil
}

func DefaultClientForAuthorization(authorization string) (*rest.Config, versioned.Interface, kubernetes.Interface, error) {
	restConfig, err := kubeconfig.GetRestConfig(authorization)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create REST config: %w", err)
	}
	wfClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failure to create workflow client: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failure to create kubernetes client: %w", err)
	}
	return restConfig, wfClient, kubeClient, nil
}
