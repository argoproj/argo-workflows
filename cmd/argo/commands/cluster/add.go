package cluster

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/config/clusters"
)

func NewAddCommand() *cobra.Command {
	return &cobra.Command{
		Use: "add CLUSTER_NAME CONTEXT_NAME",
		Example: `# Add from the current KUBECONFIG
argo cluster add agent agent

# Add from another file:

KUBECONFIG=~/.kube/config:cmd/agent/testdata/kubeconfig argo cluster add my-cluster-name my-context-name
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			clusterName := args[0]
			contextName := args[1]
			startingConfig, err := clientcmd.NewDefaultPathOptions().GetStartingConfig()
			errors.CheckError(err)
			context, ok := startingConfig.Contexts[contextName]
			if !ok {
				log.Fatalf("context named \"%s\" not found, you can list contexts with: `kubectl config get-contexts`", contextName)
			}
			c, err := clientcmd.NewDefaultClientConfig(*startingConfig, &clientcmd.ConfigOverrides{Context: *context}).ClientConfig()
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
			restConfig, err := client.GetConfig().ClientConfig()
			errors.CheckError(err)
			_, err = kubernetes.NewForConfigOrDie(restConfig).CoreV1().Secrets(client.Namespace()).
				Patch("clusters", types.MergePatchType, data)
			errors.CheckError(err)
			fmt.Printf(`added cluster named "%s" from context "%s"
`, clusterName, contextName)
		},
	}
}
