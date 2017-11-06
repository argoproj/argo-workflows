package commands

import (
	"fmt"

	"github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/executor"
	"github.com/spf13/cobra"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argoexec"
)

var (
	argoexec *executor.WorkflowExecutor

	// Global CLI flags
	GlobalArgs globalFlags
)

func init() {
	RootCmd.AddCommand(cmd.NewVersionCmd(CLIName))
}

// RootCmd is the argo root level command
var RootCmd = &cobra.Command{
	Use:   CLIName,
	Short: "argoexec is executor sidekick to workflow containers",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

type globalFlags struct {
	hostIP             string // --host-ip
	podAnnotationsPath string // --pod-annotations
}

func init() {
	RootCmd.PersistentFlags().StringVar(&GlobalArgs.hostIP, "host-ip", common.EnvVarHostIP, fmt.Sprintf("IP of host. (Default: %s)", common.EnvVarHostIP))
	RootCmd.PersistentFlags().StringVar(&GlobalArgs.podAnnotationsPath, "pod-annotations", common.PodMetadataAnnotationsPath, fmt.Sprintf("Pod annotations fiel from k8s downward API. (Default: %s)", common.PodMetadataAnnotationsPath))
}

// initExecutor is a helper to initialize the global argoexec instance
func initExecutor() {
	argoexec = &executor.WorkflowExecutor{}
}
