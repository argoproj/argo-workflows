package commands

import (
	"crypto/tls"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"reflect"
	"strconv"
	"time"

	eventsource "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	"k8s.io/utils/env"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/server/apiserver"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/util/cmd"
	"github.com/argoproj/argo-workflows/v3/util/help"
)

func NewServerCommand() *cobra.Command {
	var (
		authModes                []string
		configMap                string
		port                     int
		baseHRef                 string
		secure                   bool
		htst                     bool
		namespaced               bool   // --namespaced
		managedNamespace         string // --managed-namespace
		enableOpenBrowser        bool
		eventOperationQueueSize  int
		eventWorkerCount         int
		frameOptions             string
		accessControlAllowOrigin string
		logFormat                string // --log-format
	)

	command := cobra.Command{
		Use:   "server",
		Short: "start the Argo Server",
		Example: fmt.Sprintf(`
See %s`, help.ArgoSever),
		RunE: func(c *cobra.Command, args []string) error {
			cmd.SetLogFormatter(logFormat)
			stats.RegisterStackDumper()
			stats.StartStatsTicker(5 * time.Minute)

			config, err := client.GetConfig().ClientConfig()
			if err != nil {
				return err
			}
			version := argo.GetVersion()
			config = restclient.AddUserAgent(config, fmt.Sprintf("argo-workflows/%s argo-server", version.Version))
			config.Burst = 30
			config.QPS = 20.0

			namespace := client.Namespace()
			clients := &types.Clients{
				Workflow:    wfclientset.NewForConfigOrDie(config),
				EventSource: eventsource.NewForConfigOrDie(config),
				Sensor:      sensor.NewForConfigOrDie(config),
				Kubernetes:  kubernetes.NewForConfigOrDie(config),
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if !namespaced && managedNamespace != "" {
				log.Warn("ignoring --managed-namespace because --namespaced is false")
				managedNamespace = ""
			}
			if namespaced && managedNamespace == "" {
				managedNamespace = namespace
			}

			log.WithFields(log.Fields{
				"authModes":        authModes,
				"namespace":        namespace,
				"managedNamespace": managedNamespace,
				"baseHRef":         baseHRef,
				"secure":           secure,
			}).Info()

			var tlsConfig *tls.Config
			if secure {
				cer, err := tls.LoadX509KeyPair("argo-server.crt", "argo-server.key")
				if err != nil {
					return err
				}
				tlsMinVersion, err := env.GetInt("TLS_MIN_VERSION", tls.VersionTLS12)
				if err != nil {
					return err
				}
				tlsConfig = &tls.Config{
					Certificates:       []tls.Certificate{cer},
					InsecureSkipVerify: true,
					MinVersion:         uint16(tlsMinVersion),
				}
			} else {
				log.Warn("You are running in insecure mode. Learn how to enable transport layer security: https://argoproj.github.io/argo-workflows/tls/")
			}

			modes := auth.Modes{}
			for _, mode := range authModes {
				err := modes.Add(mode)
				if err != nil {
					return err
				}
			}
			if reflect.DeepEqual(modes, auth.Modes{auth.Server: true}) {
				log.Warn("You are running without client authentication. Learn how to enable client authentication: https://argoproj.github.io/argo-workflows/argo-server-auth-mode/")
			}

			opts := apiserver.ArgoServerOpts{
				BaseHRef:                 baseHRef,
				TLSConfig:                tlsConfig,
				HSTS:                     htst,
				Namespace:                namespace,
				Clients:                  clients,
				RestConfig:               config,
				AuthModes:                modes,
				ManagedNamespace:         managedNamespace,
				ConfigName:               configMap,
				EventOperationQueueSize:  eventOperationQueueSize,
				EventWorkerCount:         eventWorkerCount,
				XFrameOptions:            frameOptions,
				AccessControlAllowOrigin: accessControlAllowOrigin,
			}
			browserOpenFunc := func(url string) {}
			if enableOpenBrowser {
				browserOpenFunc = func(url string) {
					log.Infof("Argo UI is available at %s", url)
					err := open.Run(url)
					if err != nil {
						log.Warnf("Unable to open the browser. %v", err)
					}
				}
			}

			server, err := apiserver.NewArgoServer(ctx, opts)
			if err != nil {
				return err
			}

			// disabled by default, for security
			if x, enabled := os.LookupEnv("ARGO_SERVER_PPROF"); enabled {
				port, err := strconv.Atoi(x)
				if err != nil {
					return err
				}
				go func() {
					log.Infof("starting server for pprof on :%d, see https://golang.org/pkg/net/http/pprof/", port)
					log.Println(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
				}()
			}

			server.Run(ctx, port, browserOpenFunc)
			return nil
		},
	}

	command.Flags().IntVarP(&port, "port", "p", 2746, "Port to listen on")
	defaultBaseHRef := os.Getenv("BASE_HREF")
	if defaultBaseHRef == "" {
		defaultBaseHRef = "/"
	}
	command.Flags().StringVar(&baseHRef, "basehref", defaultBaseHRef, "Value for base href in index.html. Used if the server is running behind reverse proxy under subpath different from /. Defaults to the environment variable BASE_HREF.")
	// "-e" for encrypt, like zip
	// We default to secure mode if we find certs available, otherwise we default to insecure mode.
	_, err := os.Stat("argo-server.crt")
	command.Flags().BoolVarP(&secure, "secure", "e", !os.IsNotExist(err), "Whether or not we should listen on TLS.")
	command.Flags().BoolVar(&htst, "hsts", true, "Whether or not we should add a HTTP Secure Transport Security header. This only has effect if secure is enabled.")
	command.Flags().StringArrayVar(&authModes, "auth-mode", []string{"client"}, "API server authentication mode. Any 1 or more length permutation of: client,server,sso")
	command.Flags().StringVar(&configMap, "configmap", "workflow-controller-configmap", "Name of K8s configmap to retrieve workflow controller configuration")
	command.Flags().BoolVar(&namespaced, "namespaced", false, "run as namespaced mode")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", "", "namespace that watches, default to the installation namespace")
	command.Flags().BoolVarP(&enableOpenBrowser, "browser", "b", false, "enable automatic launching of the browser [local mode]")
	command.Flags().IntVar(&eventOperationQueueSize, "event-operation-queue-size", 16, "how many events operations that can be queued at once")
	command.Flags().IntVar(&eventWorkerCount, "event-worker-count", 4, "how many event workers to run")
	command.Flags().StringVar(&frameOptions, "x-frame-options", "DENY", "Set X-Frame-Options header in HTTP responses.")
	command.Flags().StringVar(&accessControlAllowOrigin, "access-control-allow-origin", "", "Set Access-Control-Allow-Origin header in HTTP responses.")
	command.Flags().StringVar(&logFormat, "log-format", "text", "The formatter to use for logs. One of: text|json")
	return &command
}
