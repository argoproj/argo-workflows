package client

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	"github.com/argoproj/argo-workflows/v3/util/kubeconfig"
)

var (
	ArgoServerOpts = apiclient.ArgoServerOpts{}
	instanceID     string
)

var overrides = clientcmd.ConfigOverrides{}

var (
	explicitPath string
	Offline      bool
	OfflineFiles []string
)

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
	cmd.PersistentFlags().StringVarP(&ArgoServerOpts.URL, "argo-server", "s", os.Getenv("ARGO_SERVER"), "API server `host:port`. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.")
	cmd.PersistentFlags().StringVar(&ArgoServerOpts.Path, "argo-base-href", os.Getenv("ARGO_BASE_HREF"), "Path to use with HTTP client due to Base HREF. Defaults to the ARGO_BASE_HREF environment variable.")
	cmd.PersistentFlags().BoolVar(&ArgoServerOpts.HTTP1, "argo-http1", os.Getenv("ARGO_HTTP1") == "true", "If true, use the HTTP client. Defaults to the ARGO_HTTP1 environment variable.")
	cmd.PersistentFlags().StringSliceVarP(&ArgoServerOpts.Headers, "header", "H", []string{}, "Sets additional header to all requests made by Argo CLI. (Can be repeated multiple times to add multiple headers, also supports comma separated headers) Used only when either ARGO_HTTP1 or --argo-http1 is set to true.")
	// "-e" for encrypted - like zip
	cmd.PersistentFlags().BoolVarP(&ArgoServerOpts.Secure, "secure", "e", os.Getenv("ARGO_SECURE") != "false", "Whether or not the server is using TLS with the Argo Server. Defaults to the ARGO_SECURE environment variable.")
	// "-k" like curl
	cmd.PersistentFlags().BoolVarP(&ArgoServerOpts.InsecureSkipVerify, "insecure-skip-verify", "k", os.Getenv("ARGO_INSECURE_SKIP_VERIFY") == "true", "If true, the Argo Server's certificate will not be checked for validity. This will make your HTTPS connections insecure. Defaults to the ARGO_INSECURE_SKIP_VERIFY environment variable.")
}

func NewAPIClient(ctx context.Context) (context.Context, apiclient.Client, error) {
	return apiclient.NewClientFromOpts(
		apiclient.Opts{
			ArgoServerOpts: ArgoServerOpts,
			InstanceID:     instanceID,
			AuthSupplier: func() string {
				authString, err := GetAuthString()
				if err != nil {
					log.Fatal(err)
				}
				return authString
			},
			ClientConfigSupplier: func() clientcmd.ClientConfig { return GetConfig() },
			Offline:              Offline,
			OfflineFiles:         OfflineFiles,
			Context:              ctx,
		})
}

func Namespace() string {
	if Offline {
		return ""
	}
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

func GetAuthString() (string, error) {
	token, ok := os.LookupEnv("ARGO_TOKEN")
	if ok {
		return token, nil
	}

	token = viper.GetString("token")
	if token != "" {
		return token, nil
	}

	restConfig, err := GetConfig().ClientConfig()
	if err != nil {
		return "", err
	}
	version := argo.GetVersion()
	restConfig = restclient.AddUserAgent(restConfig, fmt.Sprintf("argo-workflows/%s argo-cli", version.Version))
	authString, err := kubeconfig.GetAuthString(restConfig, explicitPath)
	if err != nil {
		return "", err
	}
	return authString, nil
}
