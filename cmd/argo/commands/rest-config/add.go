package rest_config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/config/clusters"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func NewAddCommand() *cobra.Command {
	return &cobra.Command{
		Use: "add CLUSTER_NAME.GROUP.VERSION.RESOURCE.NAMESPACE CONTEXT_NAME",
		Example: `
# whole cluster
argo rest-config add other..v1.pods. k3s-default

# just one namespace
argo rest-config add other..v1.pods.argo k3s-default

# workflows
argo rest-config add other.argoproj.io.v1alpha1.workflows.argo k3s-default
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			clusterNamespace, err := wfv1.ParseClusterNamespaceKey(args[0])
			errors.CheckError(err)
			contextName := args[1]
			startingConfig, err := clientcmd.NewDefaultPathOptions().GetStartingConfig()
			errors.CheckError(err)
			kubeContext, ok := startingConfig.Contexts[contextName]
			if !ok {
				log.Fatalf("context named \"%s\" not found, you can list contexts with: `kubectl config get-contexts`", contextName)
			}
			user := startingConfig.AuthInfos[kubeContext.AuthInfo]
			log.Debug(user)
			c, err := clientcmd.NewDefaultClientConfig(*startingConfig, &clientcmd.ConfigOverrides{Context: *kubeContext}).ClientConfig()
			errors.CheckError(err)
			log.Debug(c.String())
			data, err := json.Marshal(&clusters.Config{
				Host:               c.Host,
				APIPath:            c.APIPath,
				Username:           user.Username,
				Password:           user.Password,
				BearerToken:        user.Token,
				TLSClientConfig:    c.TLSClientConfig,
				UserAgent:          c.UserAgent,
				DisableCompression: c.DisableCompression,
				QPS:                c.QPS,
				Burst:              c.Burst,
				Timeout:            c.Timeout,
			})
			errors.CheckError(err)
			log.Debug(string(data))
			data, err = json.Marshal(map[string]map[string]string{
				"stringData": {
					string(clusterNamespace): string(data),
				},
			})
			errors.CheckError(err)
			restConfig, err := client.GetConfig().ClientConfig()
			errors.CheckError(err)
			_, err = kubernetes.NewForConfigOrDie(restConfig).CoreV1().Secrets(client.Namespace()).
				Patch(context.Background(), "rest-config", types.MergePatchType, data, metav1.PatchOptions{})
			errors.CheckError(err)
			fmt.Printf("added cluster/namespace \"%v\" from context \"%s\"\n", clusterNamespace, contextName)
		},
	}
}
