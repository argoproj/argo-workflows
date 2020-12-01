package cluster

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/cmd/argo/commands/client"
)

func NewRMCommand() *cobra.Command {
	return &cobra.Command{
		Use: "rm CLUSTER_NAME",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			clusterName := args[0]
			restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
			errors.CheckError(err)
			kube, err := kubernetes.NewForConfig(restConfig)
			errors.CheckError(err)
			secrets := kube.CoreV1().Secrets(client.Namespace())
			errors.CheckError(err)
			data, err := json.Marshal(map[string]map[string]interface{}{
				"data": {
					clusterName: nil,
				},
			})
			errors.CheckError(err)
			_, err = secrets.Patch("clusters", types.MergePatchType, data)
			errors.CheckError(err)
			fmt.Printf(`removed cluster named "%s"
`, clusterName)
		},
	}
}
