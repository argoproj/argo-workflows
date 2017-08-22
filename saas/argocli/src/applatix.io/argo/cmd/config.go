package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(configCmd)

	configCmd.AddCommand(configShowCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "config commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show details about a Argo cluster config",
	Run:   configShow,
}

func configShow(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	config := initConfig()
	const fmtStr = "%-10s %v\n"
	fmt.Printf(fmtStr, "URL:", config.URL)
	fmt.Printf(fmtStr, "Username:", config.Username)
	fmt.Printf(fmtStr, "Password:", config.Password)
	if config.Insecure != nil {
		fmt.Printf(fmtStr, "Insecure:", *config.Insecure)
	}
}
