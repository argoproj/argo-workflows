package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/argoproj/argo-workflows/v4/util/secrets"

	events "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	workflow "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/server/auth/serviceaccount"
	"github.com/argoproj/argo-workflows/v4/server/auth/sso"
	authTypes "github.com/argoproj/argo-workflows/v4/server/auth/types"
	"github.com/argoproj/argo-workflows/v4/server/cache"
	servertypes "github.com/argoproj/argo-workflows/v4/server/types"
	"github.com/argoproj/argo-workflows/v4/util/expr/argoexpr"
	jsonutil "github.com/argoproj/argo-workflows/v4/util/json"
	"github.com/argoproj/argo-workflows/v4/util/kubeconfig"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

type ContextKey string

const (
	DynamicKey ContextKey = "dynamic.Interface"
	WfKey      ContextKey = "workflow.Interface"
	EventsKey  ContextKey = "events.Interface"
	KubeKey    ContextKey = "kubernetes.Interface"
	ClaimsKey  ContextKey = "types.Claims"
)

type Gatekeeper interface {
	ContextWithRequest(ctx context.Context, req interface{}) (context.Context, error)
	Context(ctx context.Context) (context.Context, error)
	UnaryServerInterceptor() grpc.UnaryServerInterceptor
	StreamServerInterceptor() grpc.StreamServerInterceptor
}

type ClientForAuthorization func(authorization string, config *rest.Config) (*rest.Config, *servertypes.Clients, error)

type gatekeeper struct {
	Modes Modes
	// global clients, not to be used if there are better ones
	clients                *servertypes.Clients
	restConfig             *rest.Config
	ssoIf                  sso.Interface
	clientForAuthorization ClientForAuthorization
	// The namespace the server is installed in.
	namespace    string
	ssoNamespace string
	namespaced   bool
	cache        *cache.ResourceCache
}

func NewGatekeeper(modes Modes, clients *servertypes.Clients, restConfig *rest.Config, ssoIf sso.Interface, clientForAuthorization ClientForAuthorization, namespace string, ssoNamespace string, namespaced bool, cache *cache.ResourceCache) (Gatekeeper, error) {
	if len(modes) == 0 {
		return nil, fmt.Errorf("must specify at least one auth mode")
	}
	return &gatekeeper{
		modes,
		clients,
		restConfig,
		ssoIf,
		clientForAuthorization,
		namespace,
		ssoNamespace,
		namespaced,
		cache,
	}, nil

}

func (s *gatekeeper) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx, err = s.ContextWithRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (s *gatekeeper) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, NewAuthorizingServerStream(ss, s))
	}
}

func (s *gatekeeper) ContextWithRequest(ctx context.Context, req interface{}) (context.Context, error) {
	clients, claims, err := s.getClients(ctx, req)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, DynamicKey, clients.Dynamic)
	ctx = context.WithValue(ctx, WfKey, clients.Workflow)
	ctx = context.WithValue(ctx, EventsKey, clients.Events)
	ctx = context.WithValue(ctx, KubeKey, clients.Kubernetes)
	ctx = context.WithValue(ctx, ClaimsKey, claims)
	return ctx, nil
}

func (s *gatekeeper) Context(ctx context.Context) (context.Context, error) {
	return s.ContextWithRequest(ctx, nil)
}

func GetDynamicClient(ctx context.Context) dynamic.Interface {
	return ctx.Value(DynamicKey).(dynamic.Interface)
}

func GetWfClient(ctx context.Context) workflow.Interface {
	return ctx.Value(WfKey).(workflow.Interface)
}

func GetEventsClient(ctx context.Context) events.Interface {
	return ctx.Value(EventsKey).(events.Interface)
}

func GetKubeClient(ctx context.Context) kubernetes.Interface {
	return ctx.Value(KubeKey).(kubernetes.Interface)
}

func GetClaims(ctx context.Context) *authTypes.Claims {
	config, _ := ctx.Value(ClaimsKey).(*authTypes.Claims)
	return config
}

func getAuthHeaders(md metadata.MD) []string {
	// looks for the HTTP header `Authorization: Bearer ...`
	for _, t := range md.Get("authorization") {
		return []string{t}
	}
	// check the HTTP cookie
	// In cases such as wildcard domain cookies, there could be multiple authorization headers
	var authorizations []string
	for _, t := range md.Get("cookie") {
		header := http.Header{}
		header.Add("Cookie", t)
		request := http.Request{Header: header}
		cookies := request.Cookies()
		for _, c := range cookies {
			if c.Name == "authorization" {
				authorizations = append(authorizations, c.Value)
			}
		}
	}
	return authorizations
}

func (s *gatekeeper) getClients(ctx context.Context, req interface{}) (*servertypes.Clients, *authTypes.Claims, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	authorizations := getAuthHeaders(md)
	// Required for GetMode() with Server auth when no auth header specified
	if len(authorizations) == 0 {
		authorizations = append(authorizations, "")
	}
	valid := false
	var mode Mode
	var authorization string

	for _, token := range authorizations {
		mode, valid = s.Modes.GetMode(token)
		// Stop checking after the first valid token
		if valid {
			authorization = token
			break
		}
	}
	if !valid {
		return nil, nil, status.Error(codes.Unauthenticated, "token not valid. see https://argo-workflows.readthedocs.io/en/latest/faq/")
	}
	switch mode {
	case Client:
		restConfig, clients, err := s.clientForAuthorization(authorization, s.restConfig)
		if err != nil {
			return nil, nil, status.Error(codes.Unauthenticated, err.Error())
		}
		claims, _ := serviceaccount.ClaimSetFor(restConfig)
		return clients, claims, nil
	case Server:
		claims, _ := serviceaccount.ClaimSetFor(s.restConfig)
		return s.clients, claims, nil
	case SSO:
		logger := logging.RequireLoggerFromContext(ctx)
		claims, err := s.ssoIf.Authorize(authorization)
		if err != nil {
			return nil, nil, status.Error(codes.Unauthenticated, err.Error())
		}
		if s.ssoIf.IsRBACEnabled() {
			clients, err := s.rbacAuthorization(ctx, claims, req)
			if err != nil {
				logger.WithError(err).Error(ctx, "failed to perform RBAC authorization")
				return nil, nil, status.Error(codes.PermissionDenied, "not allowed")
			}
			return clients, claims, nil
		} else {
			// important! write an audit entry (i.e. log entry) so we know which user performed an operation
			logger.WithFields(addClaimsLogFields(claims, nil)).Info(ctx, "using the default service account for user")
			return s.clients, claims, nil
		}
	default:
		panic("this should never happen")
	}
}

func getNamespace(req interface{}) string {
	if req == nil {
		return ""
	}
	namespacedRequest, ok := req.(servertypes.NamespacedRequest)
	if !ok {
		return ""
	}
	return namespacedRequest.GetNamespace()
}

func precedence(serviceAccount *corev1.ServiceAccount) int {
	i, _ := strconv.Atoi(serviceAccount.Annotations[common.AnnotationKeyRBACRulePrecedence])
	return i
}

func (s *gatekeeper) getServiceAccount(claims *authTypes.Claims, namespace string) (*corev1.ServiceAccount, error) {
	list, err := s.cache.ServiceAccountLister.ServiceAccounts(namespace).List(labels.Everything())
	if err != nil {
		return nil, fmt.Errorf("failed to list SSO RBAC service accounts: %w", err)
	}
	var serviceAccounts []*corev1.ServiceAccount
	for _, serviceAccount := range list {
		_, ok := serviceAccount.Annotations[common.AnnotationKeyRBACRule]
		if !ok {
			continue
		}
		serviceAccounts = append(serviceAccounts, serviceAccount)
	}
	sort.Slice(serviceAccounts, func(i, j int) bool { return precedence(serviceAccounts[i]) > precedence(serviceAccounts[j]) })
	for _, serviceAccount := range serviceAccounts {
		rule := serviceAccount.Annotations[common.AnnotationKeyRBACRule]
		v, err := jsonutil.Jsonify(claims)
		if err != nil {
			return nil, fmt.Errorf("failed to marshall claims: %w", err)
		}
		allow, err := argoexpr.EvalBool(rule, v)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate rule: %w", err)
		}
		if !allow {
			continue
		}
		return serviceAccount, nil
	}
	return nil, fmt.Errorf("no service account rule matches")
}

func (s *gatekeeper) canDelegateRBACToRequestNamespace(req interface{}) bool {
	if s.namespaced || os.Getenv("SSO_DELEGATE_RBAC_TO_NAMESPACE") != "true" {
		return false
	}
	namespace := getNamespace(req)
	return len(namespace) != 0 && s.ssoNamespace != namespace
}

func (s *gatekeeper) getClientsForServiceAccount(ctx context.Context, claims *authTypes.Claims, serviceAccount *corev1.ServiceAccount) (*servertypes.Clients, error) {
	authorization, err := s.authorizationForServiceAccount(ctx, serviceAccount)
	if err != nil {
		return nil, err
	}
	_, clients, err := s.clientForAuthorization(authorization, s.restConfig)
	if err != nil {
		return nil, err
	}
	claims.ServiceAccountName = serviceAccount.Name
	claims.ServiceAccountNamespace = serviceAccount.Namespace
	return clients, nil
}

func (s *gatekeeper) rbacAuthorization(ctx context.Context, claims *authTypes.Claims, req interface{}) (*servertypes.Clients, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	ssoDelegationAllowed, ssoDelegated := false, false
	loginAccount, err := s.getServiceAccount(claims, s.ssoNamespace)
	if err != nil && !strings.Contains(err.Error(), "no service account rule matches") {
		return nil, err
	}
	delegatedAccount := loginAccount
	if s.canDelegateRBACToRequestNamespace(req) {
		ssoDelegationAllowed = true
		namespaceAccount, err := s.getServiceAccount(claims, getNamespace(req))
		if err != nil {
			logger.WithError(err).Info(ctx, "Error while SSO Delegation")
		} else if precedence(namespaceAccount) > precedence(loginAccount) {
			delegatedAccount = namespaceAccount
			ssoDelegated = true
		}
	}
	if delegatedAccount == nil {
		return nil, fmt.Errorf("no service account rule matches")
	}
	// important! write an audit entry (i.e. log entry) so we know which user performed an operation
	logger.WithFields(logging.Fields{
		"serviceAccount":       delegatedAccount.Name,
		"loginServiceAccount":  loginAccount.Name,
		"subject":              claims.Subject,
		"email":                claims.Email,
		"ssoDelegationAllowed": ssoDelegationAllowed,
		"ssoDelegated":         ssoDelegated,
	}).Info(ctx, "selected SSO RBAC service account for user")
	return s.getClientsForServiceAccount(ctx, claims, delegatedAccount)
}

func (s *gatekeeper) authorizationForServiceAccount(ctx context.Context, serviceAccount *corev1.ServiceAccount) (string, error) {
	secretName := secrets.TokenNameForServiceAccount(serviceAccount)
	secret, err := s.cache.GetSecret(ctx, serviceAccount.GetNamespace(), secretName)
	if err != nil {
		return "", fmt.Errorf("failed to get service account secret: %w", err)
	}
	return "Bearer " + string(secret.Data["token"]), nil
}

func addClaimsLogFields(claims *authTypes.Claims, fields logging.Fields) logging.Fields {
	if fields == nil {
		fields = logging.Fields{}
	}
	fields["subject"] = claims.Subject
	if claims.Email != "" {
		fields["email"] = claims.Email
	}
	return fields
}

func DefaultClientForAuthorization(authorization string, config *rest.Config) (*rest.Config, *servertypes.Clients, error) {
	restConfig, err := kubeconfig.GetRestConfig(authorization)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create REST config: %w", err)
	}
	restConfig = mergeServerRestConfig(config, restConfig)
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failure to create dynamic client: %w", err)
	}
	wfClient, err := workflow.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failure to create workflow client: %w", err)
	}
	eventsClient, err := events.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failure to create events client: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failure to create kubernetes client: %w", err)
	}
	return restConfig, &servertypes.Clients{
		Dynamic:    dynamicClient,
		Workflow:   wfClient,
		Events:     eventsClient,
		Kubernetes: kubeClient,
	}, nil
}

func mergeServerRestConfig(argoServerConfig *rest.Config, newConfig *rest.Config) *rest.Config {
	newConfig.Burst = argoServerConfig.Burst
	newConfig.QPS = argoServerConfig.QPS
	newConfig.UserAgent = argoServerConfig.UserAgent
	// TO DO: Merge other common configurations，such as RateLimiter.
	return newConfig
}
