package apiserver

import (
	"github.com/argoproj/argo/cmd/server/workflow"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apiclient"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/config"
	golang_proto "github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/prometheus/common/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"regexp"
	"sigs.k8s.io/yaml"

	"fmt"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"time"
)

type ArgoServer struct {
	Namespace        string
	KubeClientset    kubernetes.Clientset
	WfClientSet      *versioned.Clientset
	EnableClientAuth bool
	Config    *config.WorkflowControllerConfig
	ConfigName		 string
}

type ArgoServerOpts struct {
	Insecure         bool
	Namespace        string
	KubeClientset    *versioned.Clientset
	EnableClientAuth bool
	ConfigName		 string
}

func NewArgoServer(ctx context.Context, opts ArgoServerOpts) *ArgoServer {

	return &ArgoServer{Namespace: opts.Namespace, WfClientSet: opts.KubeClientset,
		EnableClientAuth: opts.EnableClientAuth, ConfigName:opts.ConfigName}
}

var backoff = wait.Backoff{
	Steps:    5,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

func (as *ArgoServer) Run(ctx context.Context, port int) {
	grpcs := as.newGRPCServer()
	//grpcWebS := grpcweb.WrapServer(grpcs)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 8082))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcs.Serve(lis)
}

func (as *ArgoServer) newGRPCServer() *grpc.Server {
	sOpts := []grpc.ServerOption{
		// Set the both send and receive the bytes limit to be 100MB
		// The proper way to achieve high performance is to have pagination
		// while we work toward that, we can have high limit first
		grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.MaxSendMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.ConnectionTimeout(300 * time.Second),
	}

	grpcS := grpc.NewServer(sOpts...)
	configMap, err := as.RsyncConfig(as.Namespace, as.WfClientSet, &as.KubeClientset)
	if err != nil {
		panic("Error marshalling config map")
	}
	workflowServer := workflow.NewWorkflowServer(as.Namespace, as.WfClientSet, &as.KubeClientset, configMap, as.EnableClientAuth)
	workflow.RegisterWorkflowServiceServer(grpcS, workflowServer)
	return grpcS
}

//// newHTTPServer returns the HTTP server to serve HTTP/HTTPS requests. This is implemented
//// using grpc-gateway as a proxy to the gRPC server.
//func (a *ArgoServer) newHTTPServer(ctx context.Context, port int, grpcWebHandler http.Handler) *http.KubeService {
//	endpoint := fmt.Sprintf("localhost:%d", port)
//	mux := http.NewServeMux()
//	httpS := http.KubeService{
//		Addr: endpoint,
//		Handler: &handlerSwitcher{
//			handler: &bug21955Workaround{handler: mux},
//			contentTypeToHandler: map[string]http.Handler{
//				"application/grpc-web+proto": grpcWebHandler,
//			},
//		},
//	}
//	var dOpts []grpc.DialOption
//	dOpts = append(dOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(apiclient.MaxGRPCMessageSize)))
//	//dOpts = append(dOpts, grpc.WithUserAgent(fmt.Sprintf("%s/%s", common.ArgoCDUserAgentName, argocd.GetVersion().Version)))
//
//	dOpts = append(dOpts, grpc.WithInsecure())
//
//	// HTTP 1.1+JSON Server
//	// grpc-ecosystem/grpc-gateway is used to proxy HTTP requests to the corresponding gRPC call
//	// NOTE: if a marshaller option is not supplied, grpc-gateway will default to the jsonpb from
//	// golang/protobuf. Which does not support types such as time.Time. gogo/protobuf does support
//	// time.Time, but does not support custom UnmarshalJSON() and MarshalJSON() methods. Therefore
//	//// we use our own Marshaler
//	gwMuxOpts := runtime.WithMarshalerOption(runtime.MIMEWildcard, new(jsonutil.JSONMarshaler))
//	gwCookieOpts := runtime.WithForwardResponseOption(a.translateGrpcCookieHeader)
//	gwmux := runtime.NewServeMux(gwMuxOpts, gwCookieOpts)
//	mux.Handle("/api/", gwmux)
//	mustRegisterGWHandler(workflow.RegisterWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dOpts)
//
//	return &httpS
//}
type registerFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

// mustRegisterGWHandler is a convenience function to register a gateway handler
func mustRegisterGWHandler(register registerFunc, ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) {
	err := register(ctx, mux, endpoint, opts)
	if err != nil {
		panic(err)
	}
}

type handlerSwitcher struct {
	handler              http.Handler
	contentTypeToHandler map[string]http.Handler
}

func (s *handlerSwitcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if contentHandler, ok := s.contentTypeToHandler[r.Header.Get("content-type")]; ok {
		contentHandler.ServeHTTP(w, r)
	} else {
		s.handler.ServeHTTP(w, r)
	}
}

// Workaround for https://github.com/golang/go/issues/21955 to support escaped URLs in URL path.
type bug21955Workaround struct {
	handler http.Handler
}

var pathPatters = []*regexp.Regexp{
	regexp.MustCompile(`/api/v1/clusters/[^/]+`),
	regexp.MustCompile(`/api/v1/repositories/[^/]+`),
	regexp.MustCompile(`/api/v1/repositories/[^/]+/apps`),
	regexp.MustCompile(`/api/v1/repositories/[^/]+/apps/[^/]+`),
}

func (bf *bug21955Workaround) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, pattern := range pathPatters {
		if pattern.MatchString(r.URL.RawPath) {
			r.URL.Path = r.URL.RawPath
			break
		}
	}
	bf.handler.ServeHTTP(w, r)
}

func bug21955WorkaroundInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	return handler(ctx, req)
}

// newRedirectServer returns an HTTP server which does a 307 redirect to the HTTPS server
func newRedirectServer(port int) *http.Server {
	return &http.Server{
		Addr: fmt.Sprintf("localhost:%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			target := "https://" + req.Host + req.URL.Path
			if len(req.URL.RawQuery) > 0 {
				target += "?" + req.URL.RawQuery
			}
			http.Redirect(w, req, target, http.StatusTemporaryRedirect)
		}),
	}
}

// TranslateGrpcCookieHeader conditionally sets a cookie on the response.
func (a *ArgoServer) translateGrpcCookieHeader(ctx context.Context, w http.ResponseWriter, resp golang_proto.Message) error {

	return nil
}

// ResyncConfig reloads the controller config from the configmap
func (a *ArgoServer) RsyncConfig(namespace string, wfClientset *versioned.Clientset, kubeClientSet *kubernetes.Clientset)(*config.WorkflowControllerConfig, error){
		cmClient := kubeClientSet.CoreV1().ConfigMaps(namespace)
		cm, err := cmClient.Get(a.ConfigName, metav1.GetOptions{})
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}
		return a.UpdateConfig(cm)
}

func (a *ArgoServer) UpdateConfig(cm *apiv1.ConfigMap)(*config.WorkflowControllerConfig, error){
	configStr, ok := cm.Data[common.WorkflowControllerConfigMapKey]
	if !ok {
		return nil, errors.InternalErrorf("ConfigMap '%s' does not have key '%s'", a.ConfigName, common.WorkflowControllerConfigMapKey)
	}
	var config config.WorkflowControllerConfig
	err := yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &config, nil
}

