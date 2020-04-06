package client

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/pkg/apiclient"
	"github.com/argoproj/argo/util/kubeconfig"
)

var argoServer string

var overrides = clientcmd.ConfigOverrides{}

var explicitPath string

func AddKubectlFlagsToCmd(cmd *cobra.Command) {
	kflags := clientcmd.RecommendedConfigOverrideFlags("")
	cmd.PersistentFlags().StringVar(&explicitPath, "kubeconfig", "", "Path to a kube config. Only required if out-of-cluster")
	clientcmd.BindOverrideFlags(&overrides, cmd.PersistentFlags(), kflags)
}

func GetConfig() clientcmd.ClientConfig {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = explicitPath
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
}

func AddArgoServerFlagsToCmd(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&argoServer, "argo-server", os.Getenv("ARGO_SERVER"), "API server `host:port`. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.")
}

func NewAPIClient() (context.Context, apiclient.Client) {
	ctx, client, err := apiclient.NewClient(argoServer, func() string {
		return GetAuthString()
	}, GetConfig())
	if err != nil {
		log.Fatal(err)
	}
	return ctx, client
}

func Namespace() string {
	namespace, _, err := GetConfig().Namespace()
	if err != nil {
		log.Fatal(err)
	}
	return namespace
}

func GetAuthString() string {
	restConfig, err := GetConfig().ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	authString, err := kubeconfig.GetAuthString(restConfig, explicitPath)
	if err != nil {
		log.Fatal(err)
	}
	return authString
}
