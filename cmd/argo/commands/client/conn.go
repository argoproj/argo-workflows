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

var argoServerOpts = apiclient.ArgoServerOpts{}
var offline bool

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
	// "-o" like Maven
	cmd.PersistentFlags().BoolVar(&offline, "offline", os.Getenv("ARGO_OFFLINE") == "true", "Work offline. Defaults to ARGO_OFFLINE")
	// "-s" like kubectl
	cmd.PersistentFlags().StringVarP(&argoServerOpts.URL, "argo-server", "s", os.Getenv("ARGO_SERVER"), "API server `host:port`. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.")
	// "-e" for encrypted - like zip
	cmd.PersistentFlags().BoolVarP(&argoServerOpts.Secure, "secure", "e", os.Getenv("ARGO_SECURE") == "true", "Whether or not the server is using TLS with the Argo Server. Defaults to the ARGO_SECURE environment variable.")
	// "-k" like curl
	cmd.PersistentFlags().BoolVarP(&argoServerOpts.InsecureSkipVerify, "insecure-skip-verify", "k", os.Getenv("ARGO_INSECURE_SKIP_VERIFY") == "true", "If true, the Argo Server's certificate will not be checked for validity. This will make your HTTPS connections insecure. Defaults to the ARGO_INSECURE_SKIP_VERIFY environment variable.")
}

func NewAPIClient() (context.Context, apiclient.Client) {
	opts := apiclient.Opts{
		Offline:        offline,
		ArgoServerOpts: argoServerOpts,
		AuthSupplier: func() string {
			return GetAuthString()
		},
		ClientConfigSupplier: func() clientcmd.ClientConfig {
			return GetConfig()
		},
	}
	if !offline {
		opts = apiclient.Opts{
			ClientConfig: GetConfig(),
		}
	}
	ctx, client, err := apiclient.NewClientFromOpts(opts)
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
