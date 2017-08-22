// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: fmt.Sprintf("Print the version number of %s", CLIName),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s version %s\n", CLIName, FullVersion)
	},
}
