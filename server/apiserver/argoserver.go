package apiserver

import (
	"crypto/tls"
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
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo"
	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/persist/sqldb"
	clusterwftemplatepkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	eventpkg "github.com/argoproj/argo/pkg/apiclient/event"
	infopkg "github.com/argoproj/argo/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/artifacts"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/server/auth/sso"
	"github.com/argoproj/argo/server/auth/webhook"
	"github.com/argoproj/argo/server/clusterworkflowtemplate"
	"github.com/argoproj/argo/server/cronworkflow"
	"github.com/argoproj/argo/server/event"
	"github.com/argoproj/argo/server/info"
	"github.com/argoproj/argo/server/static"
	"github.com/argoproj/argo/server/workflow"
	"github.com/argoproj/argo/server/workflowarchive"
	"github.com/argoproj/argo/server/workflowtemplate"
	grpcutil "github.com/argoproj/argo/util/grpc"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/util/json"
	"github.com/argoproj/argo/workflow/hydrator"
)

const (
	// MaxGRPCMessageSize contains max grpc message size
	MaxGRPCMessageSize = 100 * 1024 * 1024
)

type argoServer struct {
	baseHRef string
	// https://itnext.io/practical-guide-to-securing-grpc-connections-with-go-and-tls-part-1-f63058e9d6d1
	tlsConfig        *tls.Config
	hsts             bool
	namespace        string
	managedNamespace string
	kubeClientset    *kubernetes.Clientset
	wfClientSet      *versioned.Clientset
	authenticator    auth.Gatekeeper
	oAuth2Service    sso.Interface
	configController config.Controller
	stopCh           chan struct{}
	eventQueueSize   int
	eventWorkerCount int
}

type ArgoServerOpts struct {
	BaseHRef      string
	TLSConfig     *tls.Config
	Namespace     string
	KubeClientset *kubernetes.Clientset
	WfClientSet   *versioned.Clientset
	RestConfig    *rest.Config
	AuthModes     auth.Modes
	// config map name
	ConfigName              string
	ManagedNamespace        string
	HSTS                    bool
	EventOperationQueueSize int
	EventWorkerCount        int
}

func NewArgoServer(opts ArgoServerOpts) (*argoServer, error) {
	configController := config.NewController(opts.Namespace, opts.ConfigName, opts.KubeClientset)
	ssoIf := sso.NullSSO
	if opts.AuthModes[auth.SSO] {
		c, err := configController.Get()
		if err != nil {
			return nil, err
		}
		ssoIf, err = sso.New(c.SSO, opts.KubeClientset.CoreV1().Secrets(opts.Namespace), opts.BaseHRef, opts.TLSConfig != nil)
		if err != nil {
			return nil, err
		}
		log.Info("SSO enabled")
	} else {
		log.Info("SSO disabled")
	}
	gatekeeper, err := auth.NewGatekeeper(opts.AuthModes, opts.WfClientSet, opts.KubeClientset, opts.RestConfig, ssoIf)
	if err != nil {
		return nil, err
	}
	return &argoServer{
		baseHRef:         opts.BaseHRef,
		tlsConfig:        opts.TLSConfig,
		hsts:             opts.HSTS,
		namespace:        opts.Namespace,
		managedNamespace: opts.ManagedNamespace,
		wfClientSet:      opts.WfClientSet,
		kubeClientset:    opts.KubeClientset,
		authenticator:    gatekeeper,
		oAuth2Service:    ssoIf,
		configController: configController,
		stopCh:           make(chan struct{}),
		eventQueueSize:   opts.EventOperationQueueSize,
		eventWorkerCount: opts.EventWorkerCount,
	}, nil
}

var backoff = wait.Backoff{
	Steps:    5,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

func (as *argoServer) Run(ctx context.Context, port int, browserOpenFunc func(string)) {
	configMap, err := as.configController.Get()
	if err != nil {
		log.Fatal(err)
	}
	log.WithFields(log.Fields{"version": argo.GetVersion().Version, "instanceID": configMap.InstanceID}).Info("Starting Argo Server")
	instanceIDService := instanceid.NewService(configMap.InstanceID)
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
		wfArchive = sqldb.NewWorkflowArchive(session, persistence.GetClusterName(), as.managedNamespace, instanceIDService)
	}
	artifactServer := artifacts.NewArtifactServer(as.authenticator, hydrator.New(offloadRepo), wfArchive, instanceIDService)
	eventServer := event.NewController(instanceIDService, as.eventQueueSize, as.eventWorkerCount)
	grpcServer := as.newGRPCServer(instanceIDService, offloadRepo, wfArchive, eventServer, configMap.Links)
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

	if as.tlsConfig != nil {
		conn = tls.NewListener(conn, as.tlsConfig)
	}

	// Cmux is used to support servicing gRPC and HTTP1.1+JSON on the same port
	tcpm := cmux.New(conn)
	httpL := tcpm.Match(cmux.HTTP1Fast())
	grpcL := tcpm.Match(cmux.Any())

	go as.configController.Run(as.stopCh, as.restartOnConfigChange)
	go eventServer.Run(as.stopCh)
	go func() { as.checkServeErr("grpcServer", grpcServer.Serve(grpcL)) }()
	go func() { as.checkServeErr("httpServer", httpServer.Serve(httpL)) }()
	go func() { as.checkServeErr("tcpm", tcpm.Serve()) }()
	url := "http://localhost" + address
	if as.tlsConfig != nil {
		url = "https://localhost" + address
	}
	log.Infof("Argo Server started successfully on %s", url)
	browserOpenFunc(url)

	<-as.stopCh
}

func (as *argoServer) newGRPCServer(instanceIDService instanceid.Service, offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo, wfArchive sqldb.WorkflowArchive, eventServer *event.Controller, links []*v1alpha1.Link) *grpc.Server {
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

	infopkg.RegisterInfoServiceServer(grpcServer, info.NewInfoServer(as.managedNamespace, links))
	eventpkg.RegisterEventServiceServer(grpcServer, eventServer)
	workflowpkg.RegisterWorkflowServiceServer(grpcServer, workflow.NewWorkflowServer(instanceIDService, offloadNodeStatusRepo))
	workflowtemplatepkg.RegisterWorkflowTemplateServiceServer(grpcServer, workflowtemplate.NewWorkflowTemplateServer(instanceIDService))
	cronworkflowpkg.RegisterCronWorkflowServiceServer(grpcServer, cronworkflow.NewCronWorkflowServer(instanceIDService))
	workflowarchivepkg.RegisterArchivedWorkflowServiceServer(grpcServer, workflowarchive.NewWorkflowArchiveServer(wfArchive))
	clusterwftemplatepkg.RegisterClusterWorkflowTemplateServiceServer(grpcServer, clusterworkflowtemplate.NewClusterWorkflowTemplateServer(instanceIDService))
	return grpcServer
}

// newHTTPServer returns the HTTP server to serve HTTP/HTTPS requests. This is implemented
// using grpc-gateway as a proxy to the gRPC server.
func (as *argoServer) newHTTPServer(ctx context.Context, port int, artifactServer *artifacts.ArtifactServer) *http.Server {

	endpoint := fmt.Sprintf("localhost:%d", port)

	mux := http.NewServeMux()
	httpServer := http.Server{
		Addr:      endpoint,
		Handler:   mux,
		TLSConfig: as.tlsConfig,
	}
	dialOpts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxGRPCMessageSize)),
	}
	if as.tlsConfig != nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(as.tlsConfig)))
	} else {
		dialOpts = append(dialOpts, grpc.WithInsecure())
	}

	webhookInterceptor := webhook.Interceptor(as.kubeClientset)

	// HTTP 1.1+JSON Server
	// grpc-ecosystem/grpc-gateway is used to proxy HTTP requests to the corresponding gRPC call
	// NOTE: if a marshaller option is not supplied, grpc-gateway will default to the jsonpb from
	// golang/protobuf. Which does not support types such as time.Time. gogo/protobuf does support
	// time.Time, but does not support custom UnmarshalJSON() and MarshalJSON() methods. Therefore
	// we use our own Marshaler
	gwMuxOpts := runtime.WithMarshalerOption(runtime.MIMEWildcard, new(json.JSONMarshaler))
	gwmux := runtime.NewServeMux(gwMuxOpts,
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) { return key, true }),
		runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),
	)
	mustRegisterGWHandler(infopkg.RegisterInfoServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(eventpkg.RegisterEventServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowpkg.RegisterWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowtemplatepkg.RegisterWorkflowTemplateServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(cronworkflowpkg.RegisterCronWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(workflowarchivepkg.RegisterArchivedWorkflowServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(clusterwftemplatepkg.RegisterClusterWorkflowTemplateServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { webhookInterceptor(w, r, gwmux) })
	mux.HandleFunc("/artifacts/", artifactServer.GetArtifact)
	mux.HandleFunc("/artifacts-by-uid/", artifactServer.GetArtifactByUID)
	mux.HandleFunc("/oauth2/redirect", as.oAuth2Service.HandleRedirect)
	mux.HandleFunc("/oauth2/callback", as.oAuth2Service.HandleCallback)
	// we only enable HTST if we are insecure mode, otherwise you would never be able access the UI
	mux.HandleFunc("/", static.NewFilesServer(as.baseHRef, as.tlsConfig != nil && as.hsts).ServerFiles)
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
