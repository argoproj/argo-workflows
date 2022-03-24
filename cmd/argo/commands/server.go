package commands

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	eventsource "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	sensor "github.com/argoproj/argo-events/pkg/client/sensor/clientset/versioned"
	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"k8s.io/client-go/dynamic"
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
	pprofutil "github.com/argoproj/argo-workflows/v3/util/pprof"
	tlsutils "github.com/argoproj/argo-workflows/v3/util/tls"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func NewServerCommand() *cobra.Command {
	var (
		authModes                []string
		configMap                string
		port                     int
		baseHRef                 string
		secure                   bool
		tlsCertificateSecretName string
		htst                     bool
		namespaced               bool   // --namespaced
		managedNamespace         string // --managed-namespace
		ssoNamespace             string
		enableOpenBrowser        bool
		eventOperationQueueSize  int
		eventWorkerCount         int
		eventAsyncDispatch       bool
		frameOptions             string
		accessControlAllowOrigin string
		logFormat                string // --log-format
	)

	command := cobra.Command{
		Use:   "server",
		Short: "start the Argo Server",
		Example: fmt.Sprintf(`
See %s`, help.ArgoServer),
		RunE: func(c *cobra.Command, args []string) error {
			cmd.SetLogFormatter(logFormat)
			stats.RegisterStackDumper()
			stats.StartStatsTicker(5 * time.Minute)
			pprofutil.Init()

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
				Dynamic:     dynamic.NewForConfigOrDie(config),
				EventSource: eventsource.NewForConfigOrDie(config),
				Kubernetes:  kubernetes.NewForConfigOrDie(config),
				Sensor:      sensor.NewForConfigOrDie(config),
				Workflow:    wfclientset.NewForConfigOrDie(config),
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
				tlsMinVersion, err := env.GetInt("TLS_MIN_VERSION", tls.VersionTLS12)
				if err != nil {
					return err
				}

				if tlsCertificateSecretName != "" {
					log.Infof("Getting contents of Kubernetes secret %s for TLS Certificates", tlsCertificateSecretName)
					tlsConfig, err = tlsutils.GetServerTLSConfigFromSecret(ctx, clients.Kubernetes, tlsCertificateSecretName, uint16(tlsMinVersion), namespace)
					if err != nil {
						return err
					}
					log.Infof("Successfully loaded TLS config from Kubernetes secret %s", tlsCertificateSecretName)
				} else {
					log.Infof("Generating Self Signed TLS Certificates for Secure Mode")
					tlsConfig, err = tlsutils.GenerateX509KeyPairTLSConfig(uint16(tlsMinVersion))
					if err != nil {
						return err
					}
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

			if namespaced {
				// Case 1: If ssoNamespace is not specified, default it to installation namespace
				if ssoNamespace == "" {
					ssoNamespace = namespace
				}
				// Case 2: If ssoNamespace is not equal to installation or managed namespace, default it to installation namespace
				if ssoNamespace != namespace && ssoNamespace != managedNamespace {
					log.Warn("--sso-namespace should be equal to --managed-namespace or the installation namespace")
					ssoNamespace = namespace
				}
			} else {
				if ssoNamespace != "" {
					log.Warn("ignoring --sso-namespace because --namespaced is false")
				}
				ssoNamespace = namespace
			}
			opts := apiserver.ArgoServerOpts{
				BaseHRef:                 baseHRef,
				TLSConfig:                tlsConfig,
				HSTS:                     htst,
				Namespaced:               namespaced,
				Namespace:                namespace,
				SSONameSpace:             ssoNamespace,
				Clients:                  clients,
				RestConfig:               config,
				AuthModes:                modes,
				ManagedNamespace:         managedNamespace,
				ConfigName:               configMap,
				EventOperationQueueSize:  eventOperationQueueSize,
				EventWorkerCount:         eventWorkerCount,
				EventAsyncDispatch:       eventAsyncDispatch,
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

	defaultBaseHRef := os.Getenv("BASE_HREF")
	if defaultBaseHRef == "" {
		defaultBaseHRef = "/"
	}

	command.Flags().IntVarP(&port, "port", "p", 2746, "Port to listen on")
	command.Flags().StringVar(&baseHRef, "basehref", defaultBaseHRef, "Value for base href in index.html. Used if the server is running behind reverse proxy under subpath different from /. Defaults to the environment variable BASE_HREF.")
	// "-e" for encrypt, like zip
	command.Flags().BoolVarP(&secure, "secure", "e", true, "Whether or not we should listen on TLS.")
	command.Flags().BoolVar(&htst, "hsts", true, "Whether or not we should add a HTTP Secure Transport Security header. This only has effect if secure is enabled.")
	command.Flags().StringArrayVar(&authModes, "auth-mode", []string{"client"}, "API server authentication mode. Any 1 or more length permutation of: client,server,sso")
	command.Flags().StringVar(&configMap, "configmap", common.ConfigMapName, "Name of K8s configmap to retrieve workflow controller configuration")
	command.Flags().BoolVar(&namespaced, "namespaced", false, "run as namespaced mode")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", "", "namespace that watches, default to the installation namespace")
	command.Flags().StringVar(&ssoNamespace, "sso-namespace", "", "namespace that will be used for SSO RBAC. Defaults to installation namespace. Used only in namespaced mode")
	command.Flags().BoolVarP(&enableOpenBrowser, "browser", "b", false, "enable automatic launching of the browser [local mode]")
	command.Flags().IntVar(&eventOperationQueueSize, "event-operation-queue-size", 16, "how many events operations that can be queued at once")
	command.Flags().IntVar(&eventWorkerCount, "event-worker-count", 4, "how many event workers to run")
	command.Flags().BoolVar(&eventAsyncDispatch, "event-async-dispatch", false, "dispatch event async")
	command.Flags().StringVar(&frameOptions, "x-frame-options", "DENY", "Set X-Frame-Options header in HTTP responses.")
	command.Flags().StringVar(&accessControlAllowOrigin, "access-control-allow-origin", "", "Set Access-Control-Allow-Origin header in HTTP responses.")
	command.Flags().StringVar(&logFormat, "log-format", "text", "The formatter to use for logs. One of: text|json")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("ARGO")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	if err := viper.BindPFlags(command.Flags()); err != nil {
		log.Fatal(err)
	}
	command.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && viper.IsSet(f.Name) {
			val := viper.Get(f.Name)
			if err := command.Flags().Set(f.Name, fmt.Sprintf("%v", val)); err != nil {
				log.Fatal(err)
			}
		}
	})

	return &command
}
