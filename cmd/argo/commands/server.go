package commands

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"github.com/argoproj/pkg/errors"
	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/apiserver"
	"github.com/argoproj/argo/util/help"
)

func NewServerCommand() *cobra.Command {
	var (
		authMode          string
		configMap         string
		port              int
		baseHRef          string
		secure            bool
		namespaced        bool   // --namespaced
		managedNamespace  string // --managed-namespace
		enableOpenBrowser bool
	)

	var command = cobra.Command{
		Use:   "server",
		Short: "Start the Argo Server",
		Example: fmt.Sprintf(`
See %s`, help.ArgoSever),
		RunE: func(c *cobra.Command, args []string) error {
			stats.RegisterStackDumper()
			stats.StartStatsTicker(5 * time.Minute)

			config, err := client.GetConfig().ClientConfig()
			if err != nil {
				return err
			}
			config.Burst = 30
			config.QPS = 20.0

			namespace := client.Namespace()

			kubeConfig := kubernetes.NewForConfigOrDie(config)
			wflientset := wfclientset.NewForConfigOrDie(config)

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
				"authMode":         authMode,
				"namespace":        namespace,
				"managedNamespace": managedNamespace,
				"baseHRef":         baseHRef,
				"secure":           secure,
			}).Info()

			var tlsConfig *tls.Config
			if secure {
				cer, err := tls.LoadX509KeyPair("argo-server.crt", "argo-server.key")
				errors.CheckError(err)
				// InsecureSkipVerify will not impact the TLS listener. It is needed for the server to speak to itself for GRPC.
				tlsConfig = &tls.Config{Certificates: []tls.Certificate{cer}, InsecureSkipVerify: true}
			} else {
				log.Warn("You are running in insecure mode. Learn how to enable transport layer security: https://github.com/argoproj/argo/blob/master/docs/tls.md")
			}

			opts := apiserver.ArgoServerOpts{
				BaseHRef:         baseHRef,
				TLSConfig:        tlsConfig,
				Namespace:        namespace,
				WfClientSet:      wflientset,
				KubeClientset:    kubeConfig,
				RestConfig:       config,
				AuthMode:         authMode,
				ManagedNamespace: managedNamespace,
				ConfigName:       configMap,
			}
			err = opts.ValidateOpts()
			if err != nil {
				return err
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
			apiserver.NewArgoServer(opts).Run(ctx, port, browserOpenFunc)
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
	command.Flags().BoolVarP(&secure, "secure", "e", false, "Whether or not we should listen on TLS.")
	command.Flags().StringVar(&authMode, "auth-mode", "server", "API server authentication mode. One of: client|server|hybrid")
	command.Flags().StringVar(&configMap, "configmap", "workflow-controller-configmap", "Name of K8s configmap to retrieve workflow controller configuration")
	command.Flags().BoolVar(&namespaced, "namespaced", false, "run as namespaced mode")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", "", "namespace that watches, default to the installation namespace")
	command.Flags().BoolVarP(&enableOpenBrowser, "browser", "b", false, "enable automatic launching of the browser [local mode]")
	return &command
}
