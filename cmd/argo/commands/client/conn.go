package client

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/pkg/apiclient"
	"github.com/argoproj/argo/util/kubeconfig"
)

// DEPRECATED
var ArgoServer string

// DEPRECATED
var Config clientcmd.ClientConfig

var ExplicitPath string

func AddKubectlFlagsToCmd(cmd *cobra.Command) {
	// The "usual" clientcmd/kubectl flags
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	kflags := clientcmd.RecommendedConfigOverrideFlags("")
	cmd.PersistentFlags().StringVar(&ExplicitPath, "kubeconfig", "", "Path to a kube config. Only required if out-of-cluster")
	loadingRules.ExplicitPath = ExplicitPath
	clientcmd.BindOverrideFlags(&overrides, cmd.PersistentFlags(), kflags)
	Config = clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
}

func AddArgoServerFlagsToCmd(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&ArgoServer, "argo-server", os.Getenv("ARGO_SERVER"), "API server `host:port`. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.")
}

// DEPRECATED
func GetClientConn() *grpc.ClientConn {
	conn, err := apiclient.NewClientConn(ArgoServer)
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func NewAPIClient() (context.Context, apiclient.Client) {
	ctx, client, err := apiclient.NewClient(ArgoServer, func() string {
		return GetAuthString()
	}, Config)
	if err != nil {
		log.Fatal(err)
	}
	return ctx, client
}

func Namespace() string {
	namespace, _, err := Config.Namespace()
	if err != nil {
		log.Fatal(err)
	}
	return namespace
}

// DEPRECATED should only be used by client/v1 package
func GetContext() context.Context {
	return apiclient.NewContext(GetAuthString())
}

func GetAuthString() string {
	restConfig, err := Config.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	authString, err := kubeconfig.GetAuthString(restConfig, ExplicitPath)
	if err != nil {
		log.Fatal(err)
	}
	return authString
}
