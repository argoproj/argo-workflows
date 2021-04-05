package commands

import (
	"crypto/tls"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/argoproj/pkg/errors"
	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/utils/env"

	"github.com/argoproj/argo/v2/cmd/argo/commands/client"
	wfclientset "github.com/argoproj/argo/v2/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/v2/server/apiserver"
	"github.com/argoproj/argo/v2/server/auth"
	"github.com/argoproj/argo/v2/util/help"
)

func NewServerCommand() *cobra.Command {
	var (
		authModes               []string
		configMap               string
		port                    int
		baseHRef                string
		secure                  bool
		htst                    bool
		namespaced              bool   // --namespaced
		managedNamespace        string // --managed-namespace
		enableOpenBrowser       bool
		eventOperationQueueSize int
		eventWorkerCount        int
		frameOptions            string
	)

	var command = cobra.Command{
		Use:   "server",
		Short: "Start the Argo Server",
		Example: fmt.Sprintf(`
See %s`, help.ArgoSever),
		Run: func(c *cobra.Command, args []string) {
			stats.RegisterStackDumper()
			stats.StartStatsTicker(5 * time.Minute)

			config, err := client.GetConfig().ClientConfig()
			errors.CheckError(err)
			config.Burst = 30
			config.QPS = 20.0

			namespace := client.Namespace()

			kubeConfig := kubernetes.NewForConfigOrDie(config)
			wfClientSet := wfclientset.NewForConfigOrDie(config)

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
				errors.CheckError(err)
				tlsMinVersion, err := env.GetInt("TLS_MIN_VERSION", tls.VersionTLS12)
				errors.CheckError(err)
				tlsConfig = &tls.Config{
					Certificates:       []tls.Certificate{cer},
					InsecureSkipVerify: true, // InsecureSkipVerify will not impact the TLS listener. It is needed for the server to speak to itself for GRPC.
					MinVersion:         uint16(tlsMinVersion),
				}
			} else {
				log.Warn("You are running in insecure mode. Learn how to enable transport layer security: https://argoproj.github.io/argo/tls/")
			}

			modes := auth.Modes{}
			for _, mode := range authModes {
				err := modes.Add(mode)
				errors.CheckError(err)
			}
			if reflect.DeepEqual(modes, auth.Modes{auth.Server: true}) {
				log.Warn("You are running without client authentication. Learn how to enable client authentication: https://argoproj.github.io/argo/argo-server-auth-mode/")
			}

			opts := apiserver.ArgoServerOpts{
				BaseHRef:                baseHRef,
				TLSConfig:               tlsConfig,
				HSTS:                    htst,
				Namespace:               namespace,
				WfClientSet:             wfClientSet,
				KubeClientset:           kubeConfig,
				RestConfig:              config,
				AuthModes:               modes,
				ManagedNamespace:        managedNamespace,
				ConfigName:              configMap,
				EventOperationQueueSize: eventOperationQueueSize,
				EventWorkerCount:        eventWorkerCount,
				XFrameOptions:           frameOptions,
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
			server, err := apiserver.NewArgoServer(opts)
			errors.CheckError(err)
			server.Run(ctx, port, browserOpenFunc)
		},
	}

	command.Flags().IntVarP(&port, "port", "p", 2746, "Port to listen on")
	defaultBaseHRef := os.Getenv("BASE_HREF")
	if defaultBaseHRef == "" {
		defaultBaseHRef = "/"
	}
	command.Flags().StringVar(&baseHRef, "basehref", defaultBaseHRef, "Value for base href in index.html. Used if the server is running behind reverse proxy under subpath different from /. Defaults to the environment variable BASE_HREF.")
	// "-e" for encrypt, like zip
	command.Flags().BoolVarP(&secure, "secure", "e", false, "Whether or not we should listen on TLS.")
	command.Flags().BoolVar(&htst, "hsts", true, "Whether or not we should add a HTTP Secure Transport Security header. This only has effect if secure is enabled.")
	command.Flags().StringArrayVar(&authModes, "auth-mode", []string{"server"}, "API server authentication mode. Any 1 or more length permutation of: client,server,sso")
	command.Flags().StringVar(&configMap, "configmap", "workflow-controller-configmap", "Name of K8s configmap to retrieve workflow controller configuration")
	command.Flags().BoolVar(&namespaced, "namespaced", false, "run as namespaced mode")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", "", "namespace that watches, default to the installation namespace")
	command.Flags().BoolVarP(&enableOpenBrowser, "browser", "b", false, "enable automatic launching of the browser [local mode]")
	command.Flags().IntVar(&eventOperationQueueSize, "event-operation-queue-size", 16, "how many events operations that can be queued at once")
	command.Flags().IntVar(&eventWorkerCount, "event-worker-count", 4, "how many event workers to run")
	command.Flags().StringVar(&frameOptions, "x-frame-options", "DENY", "Set X-Frame-Options header in HTTP responses.")
	return &command
}
