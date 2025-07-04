package commands

import (
	"context"
	"crypto/tls"
	"fmt"
	"reflect"
	"strings"
	"time"

	events "github.com/argoproj/argo-events/pkg/client/clientset/versioned"
	"github.com/argoproj/pkg/stats"
	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
	"github.com/argoproj/argo-workflows/v3/util/logging"
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
		hsts                     bool
		namespaced               bool   // --namespaced
		managedNamespace         string // --managed-namespace
		enableOpenBrowser        bool
		eventOperationQueueSize  int
		eventWorkerCount         int
		eventAsyncDispatch       bool
		frameOptions             string
		accessControlAllowOrigin string
		apiRateLimit             uint64
		kubeAPIQPS               float32
		kubeAPIBurst             int
		allowedLinkProtocol      []string
		logFormat                string // --log-format
	)

	command := cobra.Command{
		Use:   "server",
		Short: "start the Argo Server",
		Example: fmt.Sprintf(`
See %s`, help.ArgoServer()),
		RunE: func(c *cobra.Command, args []string) error {
			cmd.SetLogFormatter(logFormat)
			ctx := c.Context()
			if ctx == nil {
				ctx = logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
				c.SetContext(ctx)
			}
			stats.RegisterStackDumper()
			stats.StartStatsTicker(5 * time.Minute)
			pprofutil.Init(ctx)

			config, err := client.GetConfig().ClientConfig()
			if err != nil {
				return err
			}
			version := argo.GetVersion()
			config = restclient.AddUserAgent(config, fmt.Sprintf("argo-workflows/%s argo-server", version.Version))
			config.Burst = kubeAPIBurst
			config.QPS = kubeAPIQPS

			namespace := client.Namespace()
			clients := &types.Clients{
				Dynamic:    dynamic.NewForConfigOrDie(config),
				Events:     events.NewForConfigOrDie(config),
				Kubernetes: kubernetes.NewForConfigOrDie(config),
				Workflow:   wfclientset.NewForConfigOrDie(config),
			}
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			if !namespaced && managedNamespace != "" {
				log.Warn("ignoring --managed-namespace because --namespaced is false")
				managedNamespace = ""
			}
			if namespaced && managedNamespace == "" {
				managedNamespace = namespace
			}

			ssoNamespace := namespace
			if managedNamespace != "" {
				ssoNamespace = managedNamespace
			}

			log.WithFields(log.Fields{
				"authModes":        authModes,
				"namespace":        namespace,
				"managedNamespace": managedNamespace,
				"ssoNamespace":     ssoNamespace,
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
				log.Warn("You are running in insecure mode. Learn how to enable transport layer security: https://argo-workflows.readthedocs.io/en/latest/tls/")
			}

			modes := auth.Modes{}
			for _, mode := range authModes {
				err := modes.Add(mode)
				if err != nil {
					return err
				}
			}
			if reflect.DeepEqual(modes, auth.Modes{auth.Server: true}) {
				log.Warn("You are running without client authentication. Learn how to enable client authentication: https://argo-workflows.readthedocs.io/en/latest/argo-server-auth-mode/")
			}

			opts := apiserver.ArgoServerOpts{
				BaseHRef:                 baseHRef,
				TLSConfig:                tlsConfig,
				HSTS:                     hsts,
				Namespaced:               namespaced,
				Namespace:                namespace,
				Clients:                  clients,
				RestConfig:               config,
				AuthModes:                modes,
				ManagedNamespace:         managedNamespace,
				SSONamespace:             ssoNamespace,
				ConfigName:               configMap,
				EventOperationQueueSize:  eventOperationQueueSize,
				EventWorkerCount:         eventWorkerCount,
				EventAsyncDispatch:       eventAsyncDispatch,
				XFrameOptions:            frameOptions,
				AccessControlAllowOrigin: accessControlAllowOrigin,
				APIRateLimit:             apiRateLimit,
				AllowedLinkProtocol:      allowedLinkProtocol,
			}
			browserOpenFunc := func(url string) {}
			if enableOpenBrowser {
				browserOpenFunc = func(url string) {
					log.Infof("Argo UI is available at %s", url)
					err := browser.OpenURL(url)
					if err != nil {
						log.Warnf("Unable to open the browser. %v", err)
					}
				}
			}

			server, err := apiserver.NewArgoServer(ctx, opts)
			if err != nil {
				return err
			}

			server.Run(ctx, port, browserOpenFunc)
			return nil
		},
	}

	command.Flags().IntVarP(&port, "port", "p", 2746, "Port to listen on")
	command.Flags().StringVar(&baseHRef, "base-href", "/", "Value for base href in index.html. Used if the server is running behind reverse proxy under subpath different from /.")
	// "-e" for encrypt, like zip
	command.Flags().BoolVarP(&secure, "secure", "e", true, "Whether or not we should listen on TLS.")
	command.Flags().StringVar(&tlsCertificateSecretName, "tls-certificate-secret-name", "", "The name of a Kubernetes secret that contains the server certificates")
	command.Flags().BoolVar(&hsts, "hsts", true, "Whether or not we should add a HTTP Secure Transport Security header. This only has effect if secure is enabled.")
	command.Flags().StringArrayVar(&authModes, "auth-mode", []string{"client"}, "API server authentication mode. Any 1 or more length permutation of: client,server,sso")
	command.Flags().StringVar(&configMap, "configmap", common.ConfigMapName, "Name of K8s configmap to retrieve workflow controller configuration")
	command.Flags().BoolVar(&namespaced, "namespaced", false, "run as namespaced mode")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", "", "namespace that watches, default to the installation namespace")
	command.Flags().BoolVarP(&enableOpenBrowser, "browser", "b", false, "enable automatic launching of the browser [local mode]")
	command.Flags().IntVar(&eventOperationQueueSize, "event-operation-queue-size", 16, "how many events operations that can be queued at once")
	command.Flags().IntVar(&eventWorkerCount, "event-worker-count", 4, "how many event workers to run")
	command.Flags().BoolVar(&eventAsyncDispatch, "event-async-dispatch", false, "dispatch event async")
	command.Flags().StringVar(&frameOptions, "x-frame-options", "DENY", "Set X-Frame-Options header in HTTP responses.")
	command.Flags().StringVar(&accessControlAllowOrigin, "access-control-allow-origin", "", "Set Access-Control-Allow-Origin header in HTTP responses.")
	command.Flags().Uint64Var(&apiRateLimit, "api-rate-limit", 1000, "Set limit per IP for api ratelimiter")
	command.Flags().StringArrayVar(&allowedLinkProtocol, "allowed-link-protocol", []string{"http", "https"}, "Allowed protocols for links feature.")
	command.Flags().StringVar(&logFormat, "log-format", "text", "The formatter to use for logs. One of: text|json")
	command.Flags().Float32Var(&kubeAPIQPS, "kube-api-qps", 20.0, "QPS to use while talking with kube-apiserver.")
	command.Flags().IntVar(&kubeAPIBurst, "kube-api-burst", 30, "Burst to use while talking with kube-apiserver.")

	// set-up env vars for the CLI such that ARGO_* env vars can be used instead of flags
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ARGO")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	// bind flags to env vars (https://github.com/spf13/viper/tree/v1.17.0#working-with-flags)
	if err := viper.BindPFlags(command.Flags()); err != nil {
		log.Fatal(err)
	}
	// workaround for handling required flags (https://github.com/spf13/viper/issues/397#issuecomment-544272457)
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
