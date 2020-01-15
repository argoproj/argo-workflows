package apiserver

import (
	"fmt"
	"net"
	"net/http"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/cmd/server/artifacts"
	"github.com/argoproj/argo/cmd/server/auth"
	"github.com/argoproj/argo/cmd/server/cronworkflow"
	"github.com/argoproj/argo/cmd/server/static"
	"github.com/argoproj/argo/cmd/server/workflow"
	"github.com/argoproj/argo/cmd/server/workflowarchive"
	"github.com/argoproj/argo/cmd/server/workflowtemplate"
	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apiclient"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	grpcutil "github.com/argoproj/argo/util/grpc"
	"github.com/argoproj/argo/util/json"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/config"
)

type argoServer struct {
	namespace     string
	kubeClientset *kubernetes.Clientset
	authenticator auth.Gatekeeper
	configName    string
	stopCh        chan struct{}
}

type ArgoServerOpts struct {
	Namespace     string
	KubeClientset *kubernetes.Clientset
	WfClientSet   *versioned.Clientset
	RestConfig    *rest.Config
	AuthMode      string
	ConfigName    string
}

func NewArgoServer(opts ArgoServerOpts) *argoServer {
	return &argoServer{
		namespace:     opts.Namespace,
		kubeClientset: opts.KubeClientset,
		authenticator: auth.NewGatekeeper(opts.AuthMode, opts.WfClientSet, opts.KubeClientset, opts.RestConfig),
		configName:    opts.ConfigName,
	}
}

var backoff = wait.Backoff{
	Steps:    5,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

func (ao ArgoServerOpts) ValidateOpts() error {
	validate := false
	for _, item := range []string{
		auth.Server,
		auth.Hybrid,
		auth.Client,
	} {
		if ao.AuthMode == item {
			validate = true
			break
		}
	}
	if !validate {
		return errors.Errorf("", "Invalid Authentication Mode. %s", ao.AuthMode)
	}
	return nil
}

func (as *argoServer) Run(ctx context.Context, port int) {

	configMap, err := as.RsyncConfig(as.namespace, as.kubeClientset)
	if err != nil {
		// TODO: this currently returns an error every time
		log.Errorf("Error marshalling config map: %s", err)
	}
	var offloadRepo sqldb.OffloadNodeStatusRepo = sqldb.ExplosiveOffloadNodeStatusRepo
	var wfArchive sqldb.WorkflowArchive = sqldb.NullWorkflowArchive
	if configMap != nil && configMap.Persistence != nil {
		session, tableName, err := sqldb.CreateDBSession(as.kubeClientset, as.namespace, configMap.Persistence)
		if err != nil {
			log.Fatal(err)
		}
		log.WithField("nodeStatusOffload", configMap.Persistence.NodeStatusOffload).Info("Offload node status")
		if configMap.Persistence.NodeStatusOffload {
			offloadRepo = sqldb.NewOffloadNodeStatusRepo(tableName, session)
		}
		wfArchive = sqldb.NewWorkflowArchive(session)
	}
	artifactServer := artifacts.NewArtifactServer(as.authenticator, offloadRepo, wfArchive)
	grpcServer := as.newGRPCServer(offloadRepo, wfArchive)
	httpServer := as.newHTTPServer(ctx, port, artifactServer)

	// Start listener
	var conn net.Listener
	var listerErr error
	address := fmt.Sprintf(":%d", port)
	err = wait.ExponentialBackoff(backoff, func() (bool, error) {
		conn, listerErr = net.Listen("tcp", address)
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
	httpL := tcpm.Match(cmux.HTTP1Fast())
	grpcL := tcpm.Match(cmux.Any())

	go func() { as.checkServeErr("grpcServer", grpcServer.Serve(grpcL)) }()
	go func() { as.checkServeErr("httpServer", httpServer.Serve(httpL)) }()
	go func() { as.checkServeErr("tcpm", tcpm.Serve()) }()
	log.Infof("Argo Server started successfully on address %s", address)
	as.stopCh = make(chan struct{})
	<-as.stopCh
}

func (as *argoServer) newGRPCServer(offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo, wfArchive sqldb.WorkflowArchive) *grpc.Server {
	serverLog := log.NewEntry(log.StandardLogger())

	sOpts := []grpc.ServerOption{
		// Set both the send and receive the bytes limit to be 100MB
		// The proper way to achieve high performance is to have pagination
		// while we work toward that, we can have high limit first
		grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.MaxSendMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.ConnectionTimeout(300 * time.Second),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_logrus.UnaryServerInterceptor(serverLog),
			grpcutil.PanicLoggerUnaryServerInterceptor(serverLog),
			grpcutil.ErrorTranslationUnaryServerInterceptor,
			as.authenticator.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_logrus.StreamServerInterceptor(serverLog),
			grpcutil.PanicLoggerStreamServerInterceptor(serverLog),
			grpcutil.ErrorTranslationStreamServerInterceptor,
			as.authenticator.StreamServerInterceptor(),
		)),
	}

	grpcServer := grpc.NewServer(sOpts...)
	workflow.RegisterWorkflowServiceServer(grpcServer, workflow.NewWorkflowServer(offloadNodeStatusRepo))
	workflowtemplate.RegisterWorkflowTemplateServiceServer(grpcServer, workflowtemplate.NewWorkflowTemplateServer())
	cronworkflow.RegisterCronWorkflowServiceServer(grpcServer, cronworkflow.NewCronWorkflowServer())
	workflowarchive.RegisterArchivedWorkflowServiceServer(grpcServer, workflowarchive.NewWorkflowArchiveServer(wfArchive))

	return grpcServer
}

// newHTTPServer returns the HTTP server to serve HTTP/HTTPS requests. This is implemented
// using grpc-gateway as a proxy to the gRPC server.
func (as *argoServer) newHTTPServer(ctx context.Context, port int, artifactServer *artifacts.ArtifactServer) *http.Server {

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
	gwmux := runtime.NewServeMux(gwMuxOpts)
	mustRegisterGWHandler(workflow.RegisterWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowtemplate.RegisterWorkflowTemplateServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(cronworkflow.RegisterCronWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowarchive.RegisterArchivedWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mux.Handle("/api/", gwmux)
	mux.HandleFunc("/artifacts/", artifactServer.GetArtifact)
	mux.HandleFunc("/artifacts-by-uid/", artifactServer.GetArtifactByUID)
	mux.HandleFunc("/", static.ServerFiles)
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

// ResyncConfig reloads the controller config from the configmap
func (as *argoServer) RsyncConfig(namespace string, kubeClientSet *kubernetes.Clientset) (*config.WorkflowControllerConfig, error) {
	cmClient := kubeClientSet.CoreV1().ConfigMaps(namespace)
	cm, err := cmClient.Get("workflow-controller-configmap", metav1.GetOptions{})
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return as.UpdateConfig(cm)
}

func (as *argoServer) UpdateConfig(cm *apiv1.ConfigMap) (*config.WorkflowControllerConfig, error) {

	configStr, ok := cm.Data[common.WorkflowControllerConfigMapKey]
	if !ok {
		return nil, errors.InternalErrorf("ConfigMap '%s' does not have key '%s'", as.configName, common.WorkflowControllerConfigMapKey)
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
func (as *argoServer) checkServeErr(name string, err error) {

	if err != nil {
		if as.stopCh == nil {
			// a nil stopCh indicates a graceful shutdown
			log.Infof("graceful shutdown %s: %v", name, err)
		} else {
			log.Fatalf("%s: %v", name, err)
		}
	} else {
		log.Infof("graceful shutdown %s", name)
	}
}
