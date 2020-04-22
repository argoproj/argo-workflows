package commands

import (
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
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/help"
)

func NewServerCommand() *cobra.Command {
	var (
		authModes         []string
		configMap         string
		port              int
		baseHRef          string
		namespaced        bool   // --namespaced
		managedNamespace  string // --managed-namespace
		enableOpenBrowser bool
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
				"authModes":        authModes,
				"namespace":        namespace,
				"managedNamespace": managedNamespace,
				"baseHRef":         baseHRef}).
				Info()

			modes := auth.Modes{}
			for _, mode := range authModes {
				err := modes.Add(auth.Mode(mode))
				if err != nil {
					log.Fatal(err)
				}
			}

			opts := apiserver.ArgoServerOpts{
				BaseHRef:         baseHRef,
				Namespace:        namespace,
				WfClientSet:      wflientset,
				KubeClientset:    kubeConfig,
				RestConfig:       config,
				AuthModes:        modes,
				ManagedNamespace: managedNamespace,
				ConfigName:       configMap,
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
	command.Flags().StringArrayVar(&authModes, "auth-mode", []string{"server"}, "API server authentication mode. One of: client|server|hybrid|sso")
	command.Flags().StringVar(&configMap, "configmap", "workflow-controller-configmap", "Name of K8s configmap to retrieve workflow controller configuration")
	command.Flags().BoolVar(&namespaced, "namespaced", false, "run as namespaced mode")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", "", "namespace that watches, default to the installation namespace")
	command.Flags().BoolVarP(&enableOpenBrowser, "browser", "b", false, "enable automatic launching of the browser [local mode]")
	return &command
}
