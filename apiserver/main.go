// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package main

import (
	"fmt"
	"os"

	"github.com/argoproj/argo/apiserver/cmd"
	"github.com/spf13/cobra"
)

// RootCmd is the argo root level command
var RootCmd = &cobra.Command{
	Use:   "argo-apiserver",
	Short: "Argo API Server",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
