package rest_config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func NewRMCommand() *cobra.Command {
	return &cobra.Command{
		Use: "rm CLUSTER_NAME/NAMESPACE",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			clusterNamespace, err := wfv1.ParseClusterNamespaceKey(args[0])
			errors.CheckError(err)
			data, err := json.Marshal(map[string]map[string]interface{}{
				"data": {
					string(clusterNamespace): nil,
				},
			})
			errors.CheckError(err)
			restConfig, err := client.GetConfig().ClientConfig()
			errors.CheckError(err)
			_, err = kubernetes.NewForConfigOrDie(restConfig).CoreV1().Secrets(client.Namespace()).
				Patch("rest-config", types.MergePatchType, data)
			errors.CheckError(err)
			fmt.Printf("removed cluster/namespace \"%v\"\n", clusterNamespace)
		},
	}
}
