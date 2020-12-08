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
	"github.com/argoproj/argo/config/clusters"
)

func NewAddCommand() *cobra.Command {
	return &cobra.Command{
		Use: "add CLUSTER_NAME [CONTEXT_NAME]",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 && len(args) > 2 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			clusterName := args[0]
			startingConfig, err := clientcmd.NewDefaultPathOptions().GetStartingConfig()
			errors.CheckError(err)
			contextName := startingConfig.CurrentContext
			if len(args) == 2 {
				contextName = args[1]
			}
			errors.CheckError(err)
			restConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
			errors.CheckError(err)
			kube, err := kubernetes.NewForConfig(restConfig)
			errors.CheckError(err)
			secrets := kube.CoreV1().Secrets(client.Namespace())
			c, err := clientcmd.NewDefaultClientConfig(*startingConfig, &clientcmd.ConfigOverrides{Context: *startingConfig.Contexts[contextName]}).ClientConfig()
			errors.CheckError(err)
			marshal, err := json.Marshal(&clusters.RestConfig{
				Host:               c.Host,
				APIPath:            c.APIPath,
				Username:           c.Username,
				Password:           c.Password,
				BearerToken:        c.BearerToken,
				TLSClientConfig:    c.TLSClientConfig,
				UserAgent:          c.UserAgent,
				DisableCompression: c.DisableCompression,
				QPS:                c.QPS,
				Burst:              c.Burst,
				Timeout:            c.Timeout,
			})
			errors.CheckError(err)
			data, err := json.Marshal(map[string]map[string]string{
				"stringData": {
					clusterName: string(marshal),
				},
			})
			errors.CheckError(err)
			_, err = secrets.Patch("clusters", types.MergePatchType, data)
			errors.CheckError(err)
			fmt.Printf(`added cluster named "%s" from context "%s"
`, contextName, clusterName)
		},
	}
}
