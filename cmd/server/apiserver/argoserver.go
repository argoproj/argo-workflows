package apiserver

import (
	"crypto/tls"
	"github.com/argoproj/argo/cmd/server/workflow"
	"github.com/argoproj/argo/cmd/server/workflowtemplate"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apiclient"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/util/json"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/config"
	golang_proto "github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"sigs.k8s.io/yaml"

	"fmt"
	"github.com/soheilhy/cmux"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"time"
)

type ArgoServer struct {
	Namespace        string
	KubeClientset    *kubernetes.Clientset
	WfClientSet      *versioned.Clientset
	EnableClientAuth bool
	Config           *config.WorkflowControllerConfig
	ConfigName       string
	stopCh           chan struct{}
}

type ArgoServerOpts struct {
	Insecure         bool
	Namespace        string
	KubeClientset    *kubernetes.Clientset
	WfClientSet      *versioned.Clientset
	EnableClientAuth bool
	ConfigName       string
}

func NewArgoServer(opts ArgoServerOpts) *ArgoServer {
	return &ArgoServer{
		Namespace:        opts.Namespace,
		WfClientSet:      opts.WfClientSet,
		KubeClientset:    opts.KubeClientset,
		EnableClientAuth: opts.EnableClientAuth,
		ConfigName:       opts.ConfigName,
	}
}

var backoff = wait.Backoff{
	Steps:    5,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

func (as *ArgoServer) useTLS() bool {
	return false
}

func (as *ArgoServer) Run(ctx context.Context, port int) {
	grpcServer := as.newGRPCServer()
	var httpServer *http.Server
	var httpsServer *http.Server
	if as.useTLS() {
		httpServer = newRedirectServer(port)
		httpsServer = as.newHTTPServer(ctx, port)
	} else {
		httpServer = as.newHTTPServer(ctx, port)
	}

	// Start listener
	var conn net.Listener
	var listerErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		conn, listerErr = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if listerErr != nil {
			log.Warnf("failed to listen: %v", listerErr)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		log.Error(err)
		return
	}

	// Cmux is used to support servicing gRPC and HTTP1.1+JSON on the same port
	tcpm := cmux.New(conn)
	var tlsm cmux.CMux
	var grpcL net.Listener
	var httpL net.Listener
	var httpsL net.Listener
	if !as.useTLS() {
		httpL = tcpm.Match(cmux.HTTP1Fast())
		grpcL = tcpm.Match(cmux.Any())
	} else {

		// If not matched, we assume that its TLS.
		tlsl := tcpm.Match(cmux.Any())
		tlsConfig := tls.Config{
			//Certificates: []tls.Certificate{*as.settings.Certificate},
		}

		tlsl = tls.NewListener(tlsl, &tlsConfig)

		// Now, we build another mux recursively to match HTTPS and gRPC.
		tlsm := cmux.New(tlsl)
		httpsL = tlsm.Match(cmux.HTTP1Fast())
		grpcL = tlsm.Match(cmux.Any())
	}

	go func() { as.checkServeErr("grpcServer", grpcServer.Serve(grpcL)) }()
	go func() { as.checkServeErr("httpServer", httpServer.Serve(httpL)) }()
	go func() { as.checkServeErr("tcpm", tcpm.Serve()) }()
	if as.useTLS() {
		go func() { as.checkServeErr("httpsServer", httpsServer.Serve(httpsL)) }()
		go func() { as.checkServeErr("tlsm", tlsm.Serve()) }()
	}
	log.Info("Argo API Server started successfully")
	as.stopCh = make(chan struct{})
	<-as.stopCh
}

func (as *ArgoServer) newGRPCServer() *grpc.Server {
	sOpts := []grpc.ServerOption{
		// Set both the send and receive the bytes limit to be 100MB
		// The proper way to achieve high performance is to have pagination
		// while we work toward that, we can have high limit first
		grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.MaxSendMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.ConnectionTimeout(300 * time.Second),
	}

	grpcServer := grpc.NewServer(sOpts...)
	configMap, err := as.RsyncConfig(as.Namespace, as.WfClientSet, as.KubeClientset)
	if err != nil {
		// TODO: this currently returns an error every time
		log.Errorf("Error marshalling config map: %s", err)
	}
	workflowServer := workflow.NewWorkflowServer(as.Namespace, as.WfClientSet, as.KubeClientset, configMap, as.EnableClientAuth)
	workflow.RegisterWorkflowServiceServer(grpcServer, workflowServer)

	workflowTemplateServer := workflowtemplate.NewWorkflowTemplateServer(as.Namespace, as.WfClientSet, as.KubeClientset, configMap, as.EnableClientAuth)
	workflowtemplate.RegisterWorkflowTemplateServiceServer(grpcServer, workflowTemplateServer)

	return grpcServer
}

// newHTTPServer returns the HTTP server to serve HTTP/HTTPS requests. This is implemented
// using grpc-gateway as a proxy to the gRPC server.
func (a *ArgoServer) newHTTPServer(ctx context.Context, port int) *http.Server {
	endpoint := fmt.Sprintf("localhost:%d", port)

	mux := http.NewServeMux()
	httpServer := http.Server{
		Addr:    endpoint,
		Handler: mux,
	}
	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(apiclient.MaxGRPCMessageSize)))
	//dialOpts = append(dialOpts, grpc.WithUserAgent(fmt.Sprintf("%s/%s", common.ArgoCDUserAgentName, argocd.GetVersion().Version)))

	dialOpts = append(dialOpts, grpc.WithInsecure())

	// HTTP 1.1+JSON Server
	// grpc-ecosystem/grpc-gateway is used to proxy HTTP requests to the corresponding gRPC call
	// NOTE: if a marshaller option is not supplied, grpc-gateway will default to the jsonpb from
	// golang/protobuf. Which does not support types such as time.Time. gogo/protobuf does support
	// time.Time, but does not support custom UnmarshalJSON() and MarshalJSON() methods. Therefore
	// we use our own Marshaler
	gwMuxOpts := runtime.WithMarshalerOption(runtime.MIMEWildcard, new(json.JSONMarshaler))
	gwCookieOpts := runtime.WithForwardResponseOption(a.translateGrpcCookieHeader)
	gwmux := runtime.NewServeMux(gwMuxOpts, gwCookieOpts)
	mustRegisterGWHandler(workflow.RegisterWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowtemplate.RegisterWorkflowTemplateServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mux.Handle("/api/", gwmux)
	return &httpServer
}

type registerFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

// mustRegisterGWHandler is a convenience function to register a gateway handler
func mustRegisterGWHandler(register registerFunc, ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) {
	err := register(ctx, mux, endpoint, opts)
	if err != nil {
		panic(err)
	}
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
func (a *ArgoServer) RsyncConfig(namespace string, wfClientset *versioned.Clientset, kubeClientSet *kubernetes.Clientset) (*config.WorkflowControllerConfig, error) {
	cmClient := kubeClientSet.CoreV1().ConfigMaps(namespace)
	cm, err := cmClient.Get("workflow-controller-configmap", metav1.GetOptions{})
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return a.UpdateConfig(cm)
}

func (a *ArgoServer) UpdateConfig(cm *apiv1.ConfigMap) (*config.WorkflowControllerConfig, error) {
	configStr, ok := cm.Data[common.WorkflowControllerConfigMapKey]
	if !ok {
		return nil, errors.InternalErrorf("ConfigMap '%s' does not have key '%s'", a.ConfigName, common.WorkflowControllerConfigMapKey)
	}
	var config config.WorkflowControllerConfig
	log.Infof("Config Map: %s", configStr)
	err := yaml.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &config, nil
}

// checkServeErr checks the error from a .Serve() call to decide if it was a graceful shutdown
func (a *ArgoServer) checkServeErr(name string, err error) {
	if err != nil {
		if a.stopCh == nil {
			// a nil stopCh indicates a graceful shutdown
			log.Infof("graceful shutdown %s: %v", name, err)
		} else {
			log.Fatalf("%s: %v", name, err)
		}
	} else {
		log.Infof("graceful shutdown %s", name)
	}
}
