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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo"
	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/persist/sqldb"
	clusterwftemplatepkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/artifacts"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/server/auth/oauth2"
	"github.com/argoproj/argo/server/clusterworkflowtemplate"
	"github.com/argoproj/argo/server/cronworkflow"
	"github.com/argoproj/argo/server/info"
	"github.com/argoproj/argo/server/rbac"
	"github.com/argoproj/argo/server/static"
	"github.com/argoproj/argo/server/workflow"
	"github.com/argoproj/argo/server/workflowarchive"
	"github.com/argoproj/argo/server/workflowtemplate"
	grpcutil "github.com/argoproj/argo/util/grpc"
	"github.com/argoproj/argo/util/json"
)

const (
	// MaxGRPCMessageSize contains max grpc message size
	MaxGRPCMessageSize = 100 * 1024 * 1024
)

type argoServer struct {
	baseHRef         string
	namespace        string
	managedNamespace string
	kubeClientset    *kubernetes.Clientset
	authenticator    *auth.Gatekeeper
	oAuth2Service    oauth2.Service
	rbacService      rbac.Service
	configController config.Controller
	stopCh           chan struct{}
}

type ArgoServerOpts struct {
	BaseHRef      string
	Namespace     string
	KubeClientset *kubernetes.Clientset
	WfClientSet   *versioned.Clientset
	RestConfig    *rest.Config
	AuthModes     auth.Modes
	// config map name
	ConfigName       string
	ManagedNamespace string
}

func NewArgoServer(opts ArgoServerOpts) (*argoServer, error) {
	service := oauth2.NullService
	if opts.AuthModes[auth.SSO] {
		secrets, err := opts.KubeClientset.CoreV1().Secrets(opts.Namespace).Get("argo-oauth2", metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		d := secrets.Data
		service, err = oauth2.NewService(string(d["issuer"]), string(d["clientId"]), string(d["clientSecret"]), string(d["redirectUrl"]), opts.BaseHRef)
		if err != nil {
			return nil, err
		}
		log.Info("SSO enabled")
	} else {
		log.Info("SSO disabled")
	}
	gatekeeper, err := auth.NewGatekeeper(opts.AuthModes, opts.WfClientSet, opts.KubeClientset, opts.RestConfig, service)
	if err != nil {
		return nil, err
	}
	configController := config.NewController(opts.Namespace, opts.ConfigName, opts.KubeClientset)
	c, err := configController.Get()
	if err != nil {
		return nil, err
	}
	rbacService := rbac.NullService
	if c.RBAC.Enabled {
		rbacService, err = rbac.NewService(c.RBAC.PolicyCSV)
		if err != nil {
			return nil, err
		}
		log.Info("RBAC enabled")
	} else {
		log.Info("RBAC disabled")
	}
	return &argoServer{
		baseHRef:         opts.BaseHRef,
		namespace:        opts.Namespace,
		managedNamespace: opts.ManagedNamespace,
		kubeClientset:    opts.KubeClientset,
		authenticator:    gatekeeper,
		oAuth2Service:    service,
		rbacService:      rbacService,
		configController: configController,
		stopCh:           make(chan struct{}),
	}, nil
}

var backoff = wait.Backoff{
	Steps:    5,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

func (as *argoServer) Run(ctx context.Context, port int, browserOpenFunc func(string)) {
	log.WithField("version", argo.GetVersion()).Info("Starting Argo Server")

	configMap, err := as.configController.Get()
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
		// we always enable node offload, as this is read-only for the Argo Server, i.e. you can turn it off if you
		// like and the controller won't offload newly created workflows, but you can still read them
		offloadRepo, err = sqldb.NewOffloadNodeStatusRepo(session, persistence.GetClusterName(), tableName)
		if err != nil {
			log.Fatal(err)
		}
		// we always enable the archive for the Argo Server, as the Argo Server does not write records, so you can
		// disable the archiving - and still read old records
		wfArchive = sqldb.NewWorkflowArchive(session, persistence.GetClusterName(), configMap.InstanceID)
	}
	artifactServer := artifacts.NewArtifactServer(as.authenticator, offloadRepo, wfArchive, as.rbacService)
	grpcServer := as.newGRPCServer(configMap.InstanceID, offloadRepo, wfArchive, configMap.Links)
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

	go as.configController.Run(as.stopCh, as.restartOnConfigChange)
	go func() { as.checkServeErr("grpcServer", grpcServer.Serve(grpcL)) }()
	go func() { as.checkServeErr("httpServer", httpServer.Serve(httpL)) }()
	go func() { as.checkServeErr("tcpm", tcpm.Serve()) }()
	log.Infof("Argo Server started successfully on address %s", address)

	browserOpenFunc("http://localhost" + address)

	<-as.stopCh
}

func (as *argoServer) newGRPCServer(instanceID string, offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo, wfArchive sqldb.WorkflowArchive, links []*v1alpha1.Link) *grpc.Server {
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
			as.rbacService.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_logrus.StreamServerInterceptor(serverLog),
			grpcutil.PanicLoggerStreamServerInterceptor(serverLog),
			grpcutil.ErrorTranslationStreamServerInterceptor,
			as.authenticator.StreamServerInterceptor(),
			as.rbacService.StreamServerInterceptor(),
		)),
	}

	grpcServer := grpc.NewServer(sOpts...)

	infopkg.RegisterInfoServiceServer(grpcServer, info.NewInfoServer(as.managedNamespace, links))
	workflowpkg.RegisterWorkflowServiceServer(grpcServer, workflow.NewWorkflowServer(instanceID, offloadNodeStatusRepo))
	workflowtemplatepkg.RegisterWorkflowTemplateServiceServer(grpcServer, workflowtemplate.NewWorkflowTemplateServer())
	cronworkflowpkg.RegisterCronWorkflowServiceServer(grpcServer, cronworkflow.NewCronWorkflowServer(instanceID))
	workflowarchivepkg.RegisterArchivedWorkflowServiceServer(grpcServer, workflowarchive.NewWorkflowArchiveServer(wfArchive))
	clusterwftemplatepkg.RegisterClusterWorkflowTemplateServiceServer(grpcServer, clusterworkflowtemplate.NewClusterWorkflowTemplateServer())
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

	dialOpts = append(dialOpts, grpc.WithInsecure())

	// HTTP 1.1+JSON Server
	// grpc-ecosystem/grpc-gateway is used to proxy HTTP requests to the corresponding gRPC call
	// NOTE: if a marshaller option is not supplied, grpc-gateway will default to the jsonpb from
	// golang/protobuf. Which does not support types such as time.Time. gogo/protobuf does support
	// time.Time, but does not support custom UnmarshalJSON() and MarshalJSON() methods. Therefore
	// we use our own Marshaler
	gwMuxOpts := runtime.WithMarshalerOption(runtime.MIMEWildcard, new(json.JSONMarshaler))
	gwmux := runtime.NewServeMux(gwMuxOpts)
	mustRegisterGWHandler(infopkg.RegisterInfoServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowpkg.RegisterWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowtemplatepkg.RegisterWorkflowTemplateServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(cronworkflowpkg.RegisterCronWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowarchivepkg.RegisterArchivedWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(clusterwftemplatepkg.RegisterClusterWorkflowTemplateServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)

	mux.Handle("/api/", gwmux)
	mux.HandleFunc("/artifacts/", artifactServer.GetArtifact)
	mux.HandleFunc("/artifacts-by-uid/", artifactServer.GetArtifactByUID)
	mux.HandleFunc("/oauth2/redirect", as.oAuth2Service.HandleRedirect)
	mux.HandleFunc("/oauth2/callback", as.oAuth2Service.HandleCallback)
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

// Unlike the controller, the server creates object based on the config map at init time, and will not pick-up on
// changes unless we restart.
// Instead of opting to re-write the server, instead we'll just listen for any old change and restart.
func (as *argoServer) restartOnConfigChange(config.Config) error {
	log.Info("config map event, exiting gracefully")
	as.stopCh <- struct{}{}
	return nil
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
