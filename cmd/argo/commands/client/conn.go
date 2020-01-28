package client

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/util/kubeconfig"
)

var ArgoServer string
var Config clientcmd.ClientConfig

func AddKubectlFlagsToCmd(cmd *cobra.Command) {
	// The "usual" clientcmd/kubectl flags
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	kflags := clientcmd.RecommendedConfigOverrideFlags("")
	cmd.PersistentFlags().StringVar(&loadingRules.ExplicitPath, "kubeconfig", "", "Path to a kube config. Only required if out-of-cluster")
	clientcmd.BindOverrideFlags(&overrides, cmd.PersistentFlags(), kflags)
	Config = clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
}

func AddArgoServerFlagsToCmd(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&ArgoServer, "argo-server", os.Getenv("ARGO_SERVER"), "API server `host:port`. e.g. localhost:2746. Defaults to the ARGO_SERVER environment variable.")
}

func GetClientConn() *grpc.ClientConn {
	conn, err := grpc.Dial(ArgoServer, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func GetContext(cmd *cobra.Command) context.Context {
	token := ""
	if cmd != nil {
		tokenVal, err := cmd.PersistentFlags().GetString("token")
		if err != nil {
			log.Fatal(err)
		}
		token = tokenVal
	}
	if len(token) == 0 {
		token = GetBearerToken()
		if len(token) == 0 {
			return context.Background()
		}
	}
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
}

func GetBearerToken() string {
	restConfig, err := Config.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	token, err := kubeconfig.GetBearerToken(restConfig)
	if err != nil {
		log.Fatal(err)
	}
	return token
}
