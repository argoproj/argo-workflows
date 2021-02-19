package auth

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/antonmedv/expr"
	eventsource "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
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

	workflow "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/server/auth/serviceaccount"
	"github.com/argoproj/argo-workflows/v3/server/auth/sso"
	"github.com/argoproj/argo-workflows/v3/server/auth/types"
	servertypes "github.com/argoproj/argo-workflows/v3/server/types"
	jsonutil "github.com/argoproj/argo-workflows/v3/util/json"
	"github.com/argoproj/argo-workflows/v3/util/kubeconfig"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type ContextKey string

const (
	WfKey          ContextKey = "workflow.Interface"
	SensorKey      ContextKey = "sensor.Interface"
	EventSourceKey ContextKey = "eventsource.Interface"
	KubeKey        ContextKey = "kubernetes.Interface"
	ClaimsKey      ContextKey = "types.Claims"
)

//go:generate mockery -name Gatekeeper

type Gatekeeper interface {
	Context(ctx context.Context) (context.Context, error)
	UnaryServerInterceptor() grpc.UnaryServerInterceptor
	StreamServerInterceptor() grpc.StreamServerInterceptor
}

type ClientForAuthorization func(authorization string) (*rest.Config, *servertypes.Clients, error)

type gatekeeper struct {
	Modes Modes
	// global clients, not to be used if there are better ones
	clients                *servertypes.Clients
	restConfig             *rest.Config
	ssoIf                  sso.Interface
	clientForAuthorization ClientForAuthorization
	// The namespace the server is installed in.
	namespace string
}

func NewGatekeeper(modes Modes, clients *servertypes.Clients, restConfig *rest.Config, ssoIf sso.Interface, clientForAuthorization ClientForAuthorization, namespace string) (Gatekeeper, error) {
	if len(modes) == 0 {
		return nil, fmt.Errorf("must specify at least one auth mode")
	}
	return &gatekeeper{modes, clients, restConfig, ssoIf, clientForAuthorization, namespace}, nil
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
	clients, claims, err := s.getClients(ctx)
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, WfKey, clients.Workflow)
	ctx = context.WithValue(ctx, EventSourceKey, clients.EventSource)
	ctx = context.WithValue(ctx, SensorKey, clients.Sensor)
	ctx = context.WithValue(ctx, KubeKey, clients.Kubernetes)
	ctx = context.WithValue(ctx, ClaimsKey, claims)
	return ctx, nil
}

func GetWfClient(ctx context.Context) workflow.Interface {
	return ctx.Value(WfKey).(workflow.Interface)
}

func GetEventSourceClient(ctx context.Context) eventsource.Interface {
	return ctx.Value(EventSourceKey).(eventsource.Interface)
}

func GetSensorClient(ctx context.Context) sensor.Interface {
	return ctx.Value(SensorKey).(sensor.Interface)
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

func (s gatekeeper) getClients(ctx context.Context) (*servertypes.Clients, *types.Claims, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	authorization := getAuthHeader(md)
	mode, valid := s.Modes.GetMode(authorization)
	if !valid {
		return nil, nil, status.Error(codes.Unauthenticated, "token not valid for running mode")
	}
	switch mode {
	case Client:
		restConfig, clients, err := s.clientForAuthorization(authorization)
		if err != nil {
			return nil, nil, status.Error(codes.Unauthenticated, err.Error())
		}
		claims, _ := serviceaccount.ClaimSetFor(restConfig)
		return clients, claims, nil
	case Server:
		claims, _ := serviceaccount.ClaimSetFor(s.restConfig)
		return s.clients, claims, nil
	case SSO:
		claims, err := s.ssoIf.Authorize(authorization)
		if err != nil {
			return nil, nil, status.Error(codes.Unauthenticated, err.Error())
		}
		if s.ssoIf.IsRBACEnabled() {
			clients, err := s.rbacAuthorization(ctx, claims)
			if err != nil {
				log.WithError(err).Error("failed to perform RBAC authorization")
				return nil, nil, status.Error(codes.PermissionDenied, "not allowed")
			}
			return clients, claims, nil
		} else {
			// important! write an audit entry (i.e. log entry) so we know which user performed an operation
			log.WithFields(log.Fields{"subject": claims.Subject}).Info("using the default service account for user")
			return s.clients, claims, nil
		}
	default:
		panic("this should never happen")
	}
}

func (s *gatekeeper) rbacAuthorization(ctx context.Context, claims *types.Claims) (*servertypes.Clients, error) {
	list, err := s.clients.Kubernetes.CoreV1().ServiceAccounts(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list SSO RBAC service accounts: %w", err)
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
		v, err := jsonutil.Jsonify(claims)
		if err != nil {
			return nil, fmt.Errorf("failed to marshall claims: %w", err)
		}
		result, err := expr.Eval(rule, v)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate rule: %w", err)
		}
		allow, ok := result.(bool)
		if !ok {
			return nil, fmt.Errorf("failed to evaluate rule: not a boolean")
		}
		if !allow {
			continue
		}
		authorization, err := s.authorizationForServiceAccount(ctx, serviceAccount.Name)
		if err != nil {
			return nil, err
		}
		_, clients, err := s.clientForAuthorization(authorization)
		if err != nil {
			return nil, err
		}
		claims.ServiceAccountName = serviceAccount.Name
		// important! write an audit entry (i.e. log entry) so we know which user performed an operation
		log.WithFields(log.Fields{"serviceAccount": serviceAccount.Name, "subject": claims.Subject}).Info("selected SSO RBAC service account for user")
		return clients, nil
	}
	return nil, fmt.Errorf("no service account rule matches")
}

func (s *gatekeeper) authorizationForServiceAccount(ctx context.Context, serviceAccountName string) (string, error) {
	serviceAccount, err := s.clients.Kubernetes.CoreV1().ServiceAccounts(s.namespace).Get(ctx, serviceAccountName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service account: %w", err)
	}
	if len(serviceAccount.Secrets) == 0 {
		return "", fmt.Errorf("expected at least one secret for SSO RBAC service account: %w", err)
	}
	secret, err := s.clients.Kubernetes.CoreV1().Secrets(s.namespace).Get(ctx, serviceAccount.Secrets[0].Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get service account secret: %w", err)
	}
	return "Bearer " + string(secret.Data["token"]), nil
}

func DefaultClientForAuthorization(authorization string) (*rest.Config, *servertypes.Clients, error) {
	restConfig, err := kubeconfig.GetRestConfig(authorization)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create REST config: %w", err)
	}
	wfClient, err := workflow.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failure to create workflow client: %w", err)
	}
	eventSourceClient, err := eventsource.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failure to create event source client: %w", err)
	}
	sensorClient, err := sensor.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failure to create sensor client: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failure to create kubernetes client: %w", err)
	}
	return restConfig, &servertypes.Clients{Workflow: wfClient, EventSource: eventSourceClient, Sensor: sensorClient, Kubernetes: kubeClient}, nil
}
