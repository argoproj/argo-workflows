package commands

import (
	"os"

	"github.com/argoproj/argo/util/cmd"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argo"
)

var (
	// Global CLI flags
	globalArgs globalFlags
)

func init() {
	RootCmd.AddCommand(cmd.NewVersionCmd(CLIName))
	addKubectlFlagsToCmd(RootCmd)
}

func addKubectlFlagsToCmd(cmd *cobra.Command) {
	// The "usual" clientcmd/kubectl flags
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := clientcmd.ConfigOverrides{}
	kflags := clientcmd.RecommendedConfigOverrideFlags("")
	cmd.PersistentFlags().StringVar(&loadingRules.ExplicitPath, "kubeconfig", "", "Path to a kube config. Only required if out-of-cluster")
	clientcmd.BindOverrideFlags(&overrides, cmd.PersistentFlags(), kflags)
	clientConfig = clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
}

type globalFlags struct {
	noColor bool // --no-color
}

// RootCmd is the argo root level command
var RootCmd = &cobra.Command{
	Use:   CLIName,
	Short: "argo is the command line interface to Argo",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}
