package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/secrets"

	events "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	workflow "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	authcookie "github.com/argoproj/argo-workflows/v4/server/auth/cookie"
	"github.com/argoproj/argo-workflows/v4/server/auth/header"
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
	ContextWithRequest(ctx context.Context, req any) (context.Context, error)
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
	headerIf               header.Interface
	clientForAuthorization ClientForAuthorization
	// The namespace the server is installed in.
	namespace    string
	ssoNamespace string
	// HeaderNamespace is where login-level RBAC rules are looked up for
	// trusted-header users. Currently the same value as SSONamespace.
	headerNamespace string
	namespaced      bool
	cache           *cache.ResourceCache
	// successful client-mode token reviews, keyed by a digest of the authorization header
	tokenReviewCache cache.Interface
}

type rbacSourceKind int

const (
	rbacSourceSSO rbacSourceKind = iota
	rbacSourceHeader
)

const (
	ssoDelegateRBACEnvVar    = "SSO_DELEGATE_RBAC_TO_NAMESPACE"
	headerDelegateRBACEnvVar = "HEADER_DELEGATE_RBAC_TO_NAMESPACE"
)

// rbacSource describes where RBAC rules for a credential source are looked up
// and which env var enables namespace delegation for it.
type rbacSource struct {
	name           string // used in log messages
	namespace      string // login-level rule namespace
	delegateEnvVar string // env var enabling namespace delegation
}

// rbacSourceFor returns the RBAC settings for the given credential source.
// SSO and header users share the same service accounts and rules, but have
// separate login namespaces and delegation switches.
func (s *gatekeeper) rbacSourceFor(kind rbacSourceKind) rbacSource {
	switch kind {
	case rbacSourceHeader:
		return rbacSource{name: "header", namespace: s.headerNamespace, delegateEnvVar: headerDelegateRBACEnvVar}
	default:
		return rbacSource{name: "SSO", namespace: s.ssoNamespace, delegateEnvVar: ssoDelegateRBACEnvVar}
	}
}

type authorizationCredential struct {
	value      string
	fromCookie bool
}

// tokenReviewCacheSize bounds the number of distinct client-mode tokens whose successful
// SelfSubjectReview is remembered.
const tokenReviewCacheSize = 1000

// errTokenNotValid is the Unauthenticated message that links users to the FAQ.
const errTokenNotValid = "token not valid. see https://argo-workflows.readthedocs.io/en/latest/faq/"

func NewGatekeeper(modes Modes, clients *servertypes.Clients, restConfig *rest.Config, ssoIf sso.Interface, headerIf header.Interface, clientForAuthorization ClientForAuthorization, namespace string, ssoNamespace string, headerNamespace string, namespaced bool, resourceCache *cache.ResourceCache, tokenReviewCacheTTL time.Duration) (Gatekeeper, error) {
	if len(modes) == 0 {
		return nil, fmt.Errorf("must specify at least one auth mode")
	}
	return &gatekeeper{
		modes,
		clients,
		restConfig,
		ssoIf,
		headerIf,
		clientForAuthorization,
		namespace,
		ssoNamespace,
		headerNamespace,
		namespaced,
		resourceCache,
		cache.NewLRUTtlCache(tokenReviewCacheTTL, tokenReviewCacheSize),
	}, nil
}

// reviewToken validates the caller's credentials with the API server via a SelfSubjectReview.
// Endpoints that never call Kubernetes would otherwise accept any token that merely looks like a
// bearer token. Successful reviews are remembered for tokenReviewCacheTTL so that the API server
// is not consulted on every request; failures are never cached.
func (s *gatekeeper) reviewToken(ctx context.Context, authorization string, kubeClient kubernetes.Interface) error {
	digest := sha256.Sum256([]byte(authorization))
	key := hex.EncodeToString(digest[:])
	if _, ok := s.tokenReviewCache.Get(key); ok {
		return nil
	}
	if _, err := kubeClient.AuthenticationV1().SelfSubjectReviews().Create(ctx, &authenticationv1.SelfSubjectReview{}, metav1.CreateOptions{}); err != nil {
		return err
	}
	s.tokenReviewCache.Add(key, struct{}{})
	return nil
}

func (s *gatekeeper) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		ctx, err = s.ContextWithRequest(ctx, req)
		if err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func (s *gatekeeper) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, NewAuthorizingServerStream(ss, s))
	}
}

func (s *gatekeeper) ContextWithRequest(ctx context.Context, req any) (context.Context, error) {
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

func getAuthHeaders(md metadata.MD) []authorizationCredential {
	// looks for the HTTP header `Authorization: Bearer ...`
	for _, t := range md.Get(authcookie.AuthorizationMetadataKey) {
		return []authorizationCredential{
			{
				value:      t,
				fromCookie: false,
			},
		}
	}
	// check the HTTP cookie
	// In cases such as wildcard domain cookies, there could be multiple authorization headers
	var authorizationCredentials []authorizationCredential
	for _, t := range md.Get("cookie") {
		header := http.Header{}
		header.Add("Cookie", t)
		request := http.Request{Header: header}
		cookies := request.Cookies()
		for _, c := range cookies {
			if c.Name == authcookie.AuthorizationCookieName {
				authorizationCredentials = append(authorizationCredentials, authorizationCredential{value: c.Value, fromCookie: true})
			}
		}
	}
	return authorizationCredentials
}

func (s *gatekeeper) getClients(ctx context.Context, req any) (*servertypes.Clients, *authTypes.Claims, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	authorizationCredentials := getAuthHeaders(md)

	//
	// Authorization Header Authentication (excluding cookie-derived credentials)
	//
	for _, authorizationCredential := range authorizationCredentials {
		if authorizationCredential.fromCookie || strings.TrimSpace(authorizationCredential.value) == "" {
			continue
		}

		if s.Modes[SSO] && strings.HasPrefix(authorizationCredential.value, sso.Prefix) {
			return s.authenticateSSO(ctx, authorizationCredential.value, req)
		}

		if s.Modes[Client] &&
			(strings.HasPrefix(authorizationCredential.value, "Bearer ") ||
				strings.HasPrefix(authorizationCredential.value, "Basic ")) {
			return s.authenticateClient(ctx, authorizationCredential.value)
		}

		// No enabled mode claimed this credential. Server auth, if enabled,
		// is the catch-all; otherwise reject.
		if s.Modes[Server] {
			return s.authenticateServer()
		}

		return nil, nil, status.Error(codes.Unauthenticated, errTokenNotValid)
	}

	//
	// Trusted Header Authentication
	//
	if s.Modes[Header] {
		clients, claims, err := s.authenticateHeader(ctx, md, req)
		if err == nil {
			return clients, claims, nil
		}
		if !errors.Is(err, authTypes.ErrNoCredentials) {
			return nil, nil, err
		}
	}

	//
	// Cookie-based Authorization Authentication
	//
	for _, authorizationCredential := range authorizationCredentials {
		if !authorizationCredential.fromCookie || strings.TrimSpace(authorizationCredential.value) == "" {
			continue
		}

		if s.Modes[SSO] && strings.HasPrefix(authorizationCredential.value, sso.Prefix) {
			return s.authenticateSSO(ctx, authorizationCredential.value, req)
		}

		if s.Modes[Client] &&
			(strings.HasPrefix(authorizationCredential.value, "Bearer ") ||
				strings.HasPrefix(authorizationCredential.value, "Basic ")) {
			return s.authenticateClient(ctx, authorizationCredential.value)
		}

		// No enabled mode claimed this cookie credential. Same rule as above.
		if s.Modes[Server] {
			return s.authenticateServer()
		}

		return nil, nil, status.Error(codes.Unauthenticated, errTokenNotValid)
	}

	if s.Modes[Server] {
		return s.authenticateServer()
	}

	return nil, nil, status.Error(codes.Unauthenticated, errTokenNotValid)
}

func (s *gatekeeper) authenticateClient(ctx context.Context, authorization string) (*servertypes.Clients, *authTypes.Claims, error) {
	restConfig, clients, err := s.clientForAuthorization(authorization, s.restConfig)
	if err != nil {
		return nil, nil, status.Error(codes.Unauthenticated, err.Error())
	}
	if err := s.reviewToken(ctx, authorization, clients.Kubernetes); err != nil {
		return nil, nil, status.Errorf(codes.Unauthenticated, "token not valid: %v", err)
	}
	claims, _ := serviceaccount.ClaimSetFor(restConfig)
	return clients, claims, nil
}

func (s *gatekeeper) authenticateServer() (*servertypes.Clients, *authTypes.Claims, error) {
	claims, _ := serviceaccount.ClaimSetFor(s.restConfig)
	return s.clients, claims, nil
}

func (s *gatekeeper) authenticateSSO(
	ctx context.Context,
	authorization string,
	req any,
) (*servertypes.Clients, *authTypes.Claims, error) {
	claims, err := s.ssoIf.Authorize(authorization)

	if err != nil {
		return nil, nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return s.authenticateClaims(ctx, s.rbacSourceFor(rbacSourceSSO), claims, req, s.ssoIf.IsRBACEnabled())
}

func (s *gatekeeper) authenticateHeader(
	ctx context.Context,
	md metadata.MD,
	req any,
) (*servertypes.Clients, *authTypes.Claims, error) {
	claims, err := s.headerIf.Authorize(md)
	if err != nil {
		if errors.Is(err, authTypes.ErrNoCredentials) {
			return nil, nil, err
		}
		return nil, nil, status.Error(codes.Unauthenticated, err.Error())
	}

	return s.authenticateClaims(ctx, s.rbacSourceFor(rbacSourceHeader), claims, req, s.headerIf.IsRBACEnabled())
}

func (s *gatekeeper) authenticateClaims(
	ctx context.Context,
	src rbacSource,
	claims *authTypes.Claims,
	req any,
	rbacEnabled bool,
) (*servertypes.Clients, *authTypes.Claims, error) {
	logger := logging.RequireLoggerFromContext(ctx)

	if rbacEnabled {
		clients, err := s.rbacAuthorization(ctx, src, claims, req)
		if err != nil {
			logger.WithError(err).Error(ctx, "failed to perform RBAC authorization")
			return nil, nil, status.Error(codes.PermissionDenied, "not allowed")
		}
		return clients, claims, nil
	}

	// important! write an audit entry (i.e. log entry) so we know which user performed an operation
	logger.WithFields(addClaimsLogFields(claims, nil)).Info(ctx, "using the default service account for user")
	return s.clients, claims, nil
}

func getNamespace(req any) string {
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
		return nil, fmt.Errorf("failed to list RBAC service accounts: %w", err)
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

func (s *gatekeeper) canDelegateRBACToRequestNamespace(src rbacSource, req any) bool {
	if s.namespaced || os.Getenv(src.delegateEnvVar) != "true" {
		return false
	}
	namespace := getNamespace(req)
	return len(namespace) != 0 && src.namespace != namespace
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

func (s *gatekeeper) rbacAuthorization(ctx context.Context, src rbacSource, claims *authTypes.Claims, req any) (*servertypes.Clients, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	delegationAllowed, delegated := false, false
	loginAccount, err := s.getServiceAccount(claims, src.namespace)
	if err != nil && !strings.Contains(err.Error(), "no service account rule matches") {
		return nil, err
	}
	delegatedAccount := loginAccount
	if s.canDelegateRBACToRequestNamespace(src, req) {
		delegationAllowed = true
		namespaceAccount, err := s.getServiceAccount(claims, getNamespace(req))
		if err != nil {
			logger.WithError(err).Info(ctx, fmt.Sprintf("Error while %s delegation", src.name))
		} else if loginAccount == nil || precedence(namespaceAccount) > precedence(loginAccount) {
			delegatedAccount = namespaceAccount
			delegated = true
		}
	}
	if delegatedAccount == nil {
		return nil, fmt.Errorf("no service account rule matches")
	}
	// important! write an audit entry (i.e. log entry) so we know which user performed an operation
	fields := logging.Fields{
		"serviceAccount":    delegatedAccount.Name,
		"subject":           claims.Subject,
		"email":             claims.Email,
		"rbacSource":        src.name,
		"delegationAllowed": delegationAllowed,
		"delegated":         delegated,
	}
	if loginAccount != nil {
		fields["loginServiceAccount"] = loginAccount.Name
	}
	logger.WithFields(fields).Info(ctx, fmt.Sprintf("selected %s RBAC service account for user", src.name))
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
