package apiserver

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	clusterwftemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	eventpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/event"
	eventsourcepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/eventsource"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	sensorpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/sensor"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/artifacts"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/auth/sso"
	"github.com/argoproj/argo-workflows/v3/server/auth/webhook"
	"github.com/argoproj/argo-workflows/v3/server/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/server/cronworkflow"
	"github.com/argoproj/argo-workflows/v3/server/event"
	"github.com/argoproj/argo-workflows/v3/server/eventsource"
	"github.com/argoproj/argo-workflows/v3/server/info"
	"github.com/argoproj/argo-workflows/v3/server/sensor"
	"github.com/argoproj/argo-workflows/v3/server/static"
	"github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/server/workflow"
	"github.com/argoproj/argo-workflows/v3/server/workflowarchive"
	"github.com/argoproj/argo-workflows/v3/server/workflowtemplate"
	grpcutil "github.com/argoproj/argo-workflows/v3/util/grpc"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/json"
	"github.com/argoproj/argo-workflows/v3/workflow/artifactrepositories"
	"github.com/argoproj/argo-workflows/v3/workflow/events"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
)

const (
	// MaxGRPCMessageSize contains max grpc message size
	MaxGRPCMessageSize = 100 * 1024 * 1024
)

type argoServer struct {
	baseHRef string
	// https://itnext.io/practical-guide-to-securing-grpc-connections-with-go-and-tls-part-1-f63058e9d6d1
	tlsConfig                *tls.Config
	hsts                     bool
	namespace                string
	managedNamespace         string
	clients                  *types.Clients
	gatekeeper               auth.Gatekeeper
	oAuth2Service            sso.Interface
	configController         config.Controller
	stopCh                   chan struct{}
	eventQueueSize           int
	eventWorkerCount         int
	xframeOptions            string
	accessControlAllowOrigin string
}

type ArgoServerOpts struct {
	BaseHRef   string
	TLSConfig  *tls.Config
	Namespace  string
	Clients    *types.Clients
	RestConfig *rest.Config
	AuthModes  auth.Modes
	// config map name
	ConfigName               string
	ManagedNamespace         string
	HSTS                     bool
	EventOperationQueueSize  int
	EventWorkerCount         int
	XFrameOptions            string
	AccessControlAllowOrigin string
}

func NewArgoServer(ctx context.Context, opts ArgoServerOpts) (*argoServer, error) {
	configController := config.NewController(opts.Namespace, opts.ConfigName, opts.Clients.Kubernetes, emptyConfigFunc)
	ssoIf := sso.NullSSO
	if opts.AuthModes[auth.SSO] {
		c, err := configController.Get(ctx)
		if err != nil {
			return nil, err
		}
		ssoIf, err = sso.New(c.(*Config).SSO, opts.Clients.Kubernetes.CoreV1().Secrets(opts.Namespace), opts.BaseHRef, opts.TLSConfig != nil)
		if err != nil {
			return nil, err
		}
		log.Info("SSO enabled")
	} else {
		log.Info("SSO disabled")
	}
	gatekeeper, err := auth.NewGatekeeper(opts.AuthModes, opts.Clients, opts.RestConfig, ssoIf, auth.DefaultClientForAuthorization, opts.Namespace)
	if err != nil {
		return nil, err
	}
	return &argoServer{
		baseHRef:                 opts.BaseHRef,
		tlsConfig:                opts.TLSConfig,
		hsts:                     opts.HSTS,
		namespace:                opts.Namespace,
		managedNamespace:         opts.ManagedNamespace,
		clients:                  opts.Clients,
		gatekeeper:               gatekeeper,
		oAuth2Service:            ssoIf,
		configController:         configController,
		stopCh:                   make(chan struct{}),
		eventQueueSize:           opts.EventOperationQueueSize,
		eventWorkerCount:         opts.EventWorkerCount,
		xframeOptions:            opts.XFrameOptions,
		accessControlAllowOrigin: opts.AccessControlAllowOrigin,
	}, nil
}

var backoff = wait.Backoff{
	Steps:    5,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

func (as *argoServer) Run(ctx context.Context, port int, browserOpenFunc func(string)) {
	v, err := as.configController.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	config := v.(*Config)
	log.WithFields(log.Fields{"version": argo.GetVersion().Version, "instanceID": config.InstanceID}).Info("Starting Argo Server")
	instanceIDService := instanceid.NewService(config.InstanceID)
	offloadRepo := sqldb.ExplosiveOffloadNodeStatusRepo
	wfArchive := sqldb.NullWorkflowArchive
	persistence := config.Persistence
	if persistence != nil {
		session, tableName, err := sqldb.CreateDBSession(as.clients.Kubernetes, as.namespace, persistence)
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
	eventRecorderManager := events.NewEventRecorderManager(as.clients.Kubernetes)
	artifactRepositories := artifactrepositories.New(as.clients.Kubernetes, as.managedNamespace, &config.ArtifactRepository)
	artifactServer := artifacts.NewArtifactServer(as.gatekeeper, hydrator.New(offloadRepo), wfArchive, instanceIDService, artifactRepositories)
	eventServer := event.NewController(instanceIDService, eventRecorderManager, as.eventQueueSize, as.eventWorkerCount)
	grpcServer := as.newGRPCServer(instanceIDService, offloadRepo, wfArchive, eventServer, config.Links)
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

	// "Prometheus histograms are a great way to measure latency distributions of your RPCs. However, since it is bad practice to have metrics of high cardinality the latency monitoring metrics are disabled by default. To enable them please call the following in your server initialization code:"
	grpc_prometheus.EnableHandlingTimeHistogram()

	sOpts := []grpc.ServerOption{
		// Set both the send and receive the bytes limit to be 100MB
		// The proper way to achieve high performance is to have pagination
		// while we work toward that, we can have high limit first
		grpc.MaxRecvMsgSize(MaxGRPCMessageSize),
		grpc.MaxSendMsgSize(MaxGRPCMessageSize),
		grpc.ConnectionTimeout(300 * time.Second),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_prometheus.UnaryServerInterceptor,
			grpc_logrus.UnaryServerInterceptor(serverLog),
			grpcutil.PanicLoggerUnaryServerInterceptor(serverLog),
			grpcutil.ErrorTranslationUnaryServerInterceptor,
			as.gatekeeper.UnaryServerInterceptor(),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_prometheus.StreamServerInterceptor,
			grpc_logrus.StreamServerInterceptor(serverLog),
			grpcutil.PanicLoggerStreamServerInterceptor(serverLog),
			grpcutil.ErrorTranslationStreamServerInterceptor,
			as.gatekeeper.StreamServerInterceptor(),
		)),
	}

	grpcServer := grpc.NewServer(sOpts...)

	infopkg.RegisterInfoServiceServer(grpcServer, info.NewInfoServer(as.managedNamespace, links))
	eventpkg.RegisterEventServiceServer(grpcServer, eventServer)
	eventsourcepkg.RegisterEventSourceServiceServer(grpcServer, eventsource.NewEventSourceServer())
	sensorpkg.RegisterSensorServiceServer(grpcServer, sensor.NewSensorServer())
	workflowpkg.RegisterWorkflowServiceServer(grpcServer, workflow.NewWorkflowServer(instanceIDService, offloadNodeStatusRepo))
	workflowtemplatepkg.RegisterWorkflowTemplateServiceServer(grpcServer, workflowtemplate.NewWorkflowTemplateServer(instanceIDService))
	cronworkflowpkg.RegisterCronWorkflowServiceServer(grpcServer, cronworkflow.NewCronWorkflowServer(instanceIDService))
	workflowarchivepkg.RegisterArchivedWorkflowServiceServer(grpcServer, workflowarchive.NewWorkflowArchiveServer(wfArchive))
	clusterwftemplatepkg.RegisterClusterWorkflowTemplateServiceServer(grpcServer, clusterworkflowtemplate.NewClusterWorkflowTemplateServer(instanceIDService))
	grpc_prometheus.Register(grpcServer)
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
		grpc.WithInsecure(),
	}
	webhookInterceptor := webhook.Interceptor(as.clients.Kubernetes)

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
	mustRegisterGWHandler(eventsourcepkg.RegisterEventSourceServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
	mustRegisterGWHandler(sensorpkg.RegisterSensorServiceHandlerFromEndpoint, ctx, gwmux, endpoint, dialOpts)
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
	mux.Handle("/metrics", promhttp.Handler())
	// we only enable HTST if we are secure mode, otherwise you would never be able access the UI
	mux.HandleFunc("/", static.NewFilesServer(as.baseHRef, as.tlsConfig != nil && as.hsts, as.xframeOptions, as.accessControlAllowOrigin).ServerFiles)
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
func (as *argoServer) restartOnConfigChange(interface{}) error {
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
