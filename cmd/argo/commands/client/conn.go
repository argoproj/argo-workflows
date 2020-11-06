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
var instanceID string

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

func AddAPIClientFlagsToCmd(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&instanceID, "instanceid", os.Getenv("ARGO_INSTANCEID"), "submit with a specific controller's instance id label. Default to the ARGO_INSTANCEID environment variable.")
	// "-s" like kubectl
	cmd.PersistentFlags().StringVarP(&argoServerOpts.URL, "argo-server", "s", os.Getenv("ARGO_SERVER"), "API server `host:port`. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.")
	cmd.PersistentFlags().StringVar(&argoServerOpts.Path, "argo-base-href", os.Getenv("ARGO_BASE_HREF"), "An path to use with HTTP client (e.g. due to BASE_HREF). Defaults to the ARGO_BASE_HREF environment variable.")
	cmd.PersistentFlags().BoolVar(&argoServerOpts.HTTP1, "argo-http1", os.Getenv("ARGO_HTTP1") == "true", "If true, use the HTTP client. Defaults to the ARGO_HTTP1 environment variable.")
	// "-e" for encrypted - like zip
	cmd.PersistentFlags().BoolVarP(&argoServerOpts.Secure, "secure", "e", os.Getenv("ARGO_SECURE") == "true", "Whether or not the server is using TLS with the Argo Server. Defaults to the ARGO_SECURE environment variable.")
	// "-k" like curl
	cmd.PersistentFlags().BoolVarP(&argoServerOpts.InsecureSkipVerify, "insecure-skip-verify", "k", os.Getenv("ARGO_INSECURE_SKIP_VERIFY") == "true", "If true, the Argo Server's certificate will not be checked for validity. This will make your HTTPS connections insecure. Defaults to the ARGO_INSECURE_SKIP_VERIFY environment variable.")
}

func NewAPIClient() (context.Context, apiclient.Client) {
	ctx, client, err := apiclient.NewClientFromOpts(
		apiclient.Opts{
			ArgoServerOpts: argoServerOpts,
			InstanceID:     instanceID,
			AuthSupplier: func() string {
				return GetAuthString()
			},
			ClientConfigSupplier: func() clientcmd.ClientConfig { return GetConfig() },
		})
	if err != nil {
		log.Fatal(err)
	}
	return ctx, client
}

func Namespace() string {
	if overrides.Context.Namespace != "" {
		return overrides.Context.Namespace
	}
	namespace, ok := os.LookupEnv("ARGO_NAMESPACE")
	if ok {
		return namespace
	}
	namespace, _, err := GetConfig().Namespace()
	if err != nil {
		log.Fatal(err)
	}
	return namespace
}

func GetAuthString() string {
	token, ok := os.LookupEnv("ARGO_TOKEN")
	if ok {
		return token
	}
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
