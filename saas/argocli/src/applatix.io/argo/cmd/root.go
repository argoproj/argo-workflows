// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package cmd

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "argo",
	Short: "argo is the command line interface to Argo clusters",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}
