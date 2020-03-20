package client

import (
	"context"
	"fmt"
	"os"
	"os/user"

	"github.com/argoproj/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/pkg/apiclient"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/kubeconfig"
)

var argoServer string

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
	cmd.PersistentFlags().StringVar(&argoServer, "argo-server", os.Getenv("ARGO_SERVER"), "API server `host:port`. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.")
}

func NewAPIClient() (context.Context, apiclient.Client) {
	ctx, client, err := apiclient.NewClient(argoServer, func() string {
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

func GetUser() wfv1.User {
	current, err := user.Current()
	errors.CheckError(err)
	return wfv1.User{Name: current.Username}
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
	return fmt.Sprintf("v2/%s/%s", authString, GetUser().Name)
}
