package commands

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(sidecarCmd)
	sidecarCmd.AddCommand(sidecarWaitCmd)

}

var sidecarCmd = &cobra.Command{
	Use:   "sidecar",
	Short: "Sidecar commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

var sidecarWaitCmd = &cobra.Command{
	Use:   "wait",
	Short: "wait for user container to finish",
	Run:   waitContainer,
}

func waitContainer(cmd *cobra.Command, args []string) {
	wfExecutor := initExecutor()
	// Wait for main container to be ready
	err := wfExecutor.WaitForReady()

	// Todo: Decide wether to always exit non-zero if error happens in the sidecar
	if err != nil {
		log.Errorf("Error waiting on main container to be ready, %+v", err)
	}
	os.Exit(0)
}
