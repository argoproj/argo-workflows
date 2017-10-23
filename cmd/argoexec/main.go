package main

import (
	"fmt"
	"os"

	"github.com/argoproj/argo/util/cmd"
	"github.com/spf13/cobra"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argoexec"
)

// RootCmd is the root level command
var RootCmd = &cobra.Command{
	Use:   CLIName,
	Short: "Argo Executor",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

func init() {
	RootCmd.AddCommand(cmd.NewVersionCmd(CLIName))
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
