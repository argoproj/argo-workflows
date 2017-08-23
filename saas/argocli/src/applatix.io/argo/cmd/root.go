// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package cmd

import (
	"applatix.io/axops/utils"
	"github.com/spf13/cobra"
)

// RootCmd is the argo root level command
var RootCmd = &cobra.Command{
	Use:   "argo",
	Short: "argo is the command line interface to Argo clusters",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

func init() {
	cobra.OnInitialize(initializeSession)
}

func initializeSession() {
	jobStatusIconMap = map[int]string{
		utils.ServiceStatusInitiating: ansiFormat("⧖", noFormat),
		utils.ServiceStatusWaiting:    ansiFormat("⧖", noFormat),
		utils.ServiceStatusRunning:    ansiFormat("●", FgCyan),
		utils.ServiceStatusCanceling:  ansiFormat("⚠", FgYellow),
		utils.ServiceStatusCancelled:  ansiFormat("⚠", FgYellow),
		utils.ServiceStatusSkipped:    ansiFormat("-", noFormat),
		utils.ServiceStatusSuccess:    ansiFormat("✔", FgGreen),
		utils.ServiceStatusFailed:     ansiFormat("✖", FgRed),
	}
}
