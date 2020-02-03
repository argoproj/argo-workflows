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
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/persist/sqldb"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/artifacts"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/server/cronworkflow"
	"github.com/argoproj/argo/server/info"
	"github.com/argoproj/argo/server/static"
	"github.com/argoproj/argo/server/workflow"
	"github.com/argoproj/argo/server/workflowarchive"
	"github.com/argoproj/argo/server/workflowtemplate"
	grpcutil "github.com/argoproj/argo/util/grpc"
	"github.com/argoproj/argo/util/json"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/config"
)

const (
	// MaxGRPCMessageSize contains max grpc message size
	MaxGRPCMessageSize   = 100 * 1024 * 1024
	// Default listening host
	DefaultListeningHost = "127.0.0.1"
)

type argoServer struct {
	baseHRef         string
	namespace        string
	managedNamespace string
	kubeClientset    *kubernetes.Clientset
	authenticator    auth.Gatekeeper
	configName       string
	stopCh           chan struct{}
}

type ArgoServerOpts struct {
	BaseHRef      string
	Namespace     string
	KubeClientset *kubernetes.Clientset
	WfClientSet   *versioned.Clientset
	RestConfig    *rest.Config
	AuthMode      string
	// config map name
	ConfigName       string
	ManagedNamespace string
}

func NewArgoServer(opts ArgoServerOpts) *argoServer {
	return &argoServer{
		baseHRef:         opts.BaseHRef,
		namespace:        opts.Namespace,
		managedNamespace: opts.ManagedNamespace,
		kubeClientset:    opts.KubeClientset,
		authenticator:    auth.NewGatekeeper(opts.AuthMode, opts.WfClientSet, opts.KubeClientset, opts.RestConfig),
		configName:       opts.ConfigName,
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

func (as *argoServer) Run(ctx context.Context, port int, browserOpenFunc func(string)) {

	configMap, err := as.rsyncConfig()
	if err != nil {
		log.Fatal(err)
	}
	err = as.restartOnConfigChange(ctx.Done())
	if err != nil {
		log.Fatal(err)
	}
	var offloadRepo = sqldb.ExplosiveOffloadNodeStatusRepo
	var wfArchive = sqldb.NullWorkflowArchive
	persistence := configMap.Persistence
	if persistence != nil {
		session, tableName, err := sqldb.CreateDBSession(as.kubeClientset, as.namespace, persistence)
		if err != nil {
			log.Fatal(err)
		}
		log.WithField("nodeStatusOffload", persistence.NodeStatusOffload).Info("Offload node status")
		if persistence.NodeStatusOffload {
			offloadRepo = sqldb.NewOffloadNodeStatusRepo(session, persistence.GetClusterName(), tableName)
		}
		wfArchive = sqldb.NewWorkflowArchive(session, persistence.GetClusterName())
	}
	artifactServer := artifacts.NewArtifactServer(as.authenticator, offloadRepo, wfArchive)
	grpcServer := as.newGRPCServer(offloadRepo, wfArchive)
	httpServer := as.newHTTPServer(ctx, port, artifactServer)

	// Start listener
	var conn net.Listener
	var listerErr error
	address := fmt.Sprintf("%s:%d", DefaultListeningHost, port)
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
	log.Infof("Argo Server started successfully on address %s", conn.Addr().String())

	browserOpenFunc("http://" + conn.Addr().String())

	as.stopCh = make(chan struct{})
	<-as.stopCh
}

func (as *argoServer) newGRPCServer(offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo, wfArchive sqldb.WorkflowArchive) *grpc.Server {
	serverLog := log.NewEntry(log.StandardLogger())

	sOpts := []grpc.ServerOption{
		// Set both the send and receive the bytes limit to be 100MB
		// The proper way to achieve high performance is to have pagination
		// while we work toward that, we can have high limit first
		grpc.MaxRecvMsgSize(MaxGRPCMessageSize),
		grpc.MaxSendMsgSize(MaxGRPCMessageSize),
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

	info.RegisterInfoServiceServer(grpcServer, info.NewInfoServer(as.managedNamespace))
	workflowpkg.RegisterWorkflowServiceServer(grpcServer, workflow.NewWorkflowServer(offloadNodeStatusRepo))
	workflowtemplatepkg.RegisterWorkflowTemplateServiceServer(grpcServer, workflowtemplate.NewWorkflowTemplateServer())
	cronworkflowpkg.RegisterCronWorkflowServiceServer(grpcServer, cronworkflow.NewCronWorkflowServer())
	workflowarchivepkg.RegisterArchivedWorkflowServiceServer(grpcServer, workflowarchive.NewWorkflowArchiveServer(wfArchive))

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
	dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxGRPCMessageSize)))
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
	mustRegisterGWHandler(info.RegisterInfoServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowpkg.RegisterWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowtemplatepkg.RegisterWorkflowTemplateServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(cronworkflowpkg.RegisterCronWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowarchivepkg.RegisterArchivedWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mux.Handle("/api/", gwmux)
	mux.HandleFunc("/artifacts/", artifactServer.GetArtifact)
	mux.HandleFunc("/artifacts-by-uid/", artifactServer.GetArtifactByUID)
	mux.HandleFunc("/", static.NewFilesServer(as.baseHRef).ServerFiles)
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
func (as *argoServer) rsyncConfig() (*config.WorkflowControllerConfig, error) {
	cm, err := as.kubeClientset.CoreV1().ConfigMaps(as.namespace).
		Get(as.configName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return as.updateConfig(cm)
}

// Unlike the controller, the server creates object based on the config map at init time, and will not pick-up on
// changes unless we restart.
// Instead of opting to re-write the server, instead we'll just listen for any old change and restart.
func (as *argoServer) restartOnConfigChange(stopCh <-chan struct{}) error {
	w, err := as.kubeClientset.CoreV1().ConfigMaps(as.namespace).
		Watch(metav1.ListOptions{FieldSelector: "metadata.name=" + as.configName})
	if err != nil {
		return err
	}
	go func() {
		defer w.Stop()
		for {
			select {
			// normal exit, e.g. due to user interupt
			case <-stopCh:
				return
			case e := <-w.ResultChan():
				if e.Type != watch.Added {
					log.WithField("eventType", e.Type).Info("config map event, exiting gracefully")
					as.stopCh <- struct{}{}
					return
				}
			}
		}
	}()
	return nil
}

func (as *argoServer) updateConfig(cm *apiv1.ConfigMap) (*config.WorkflowControllerConfig, error) {

	configStr, ok := cm.Data[common.WorkflowControllerConfigMapKey]
	if !ok {
		log.Warnf("ConfigMap '%s' does not have key '%s'", as.configName, common.WorkflowControllerConfigMapKey)
		configStr = ""
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
