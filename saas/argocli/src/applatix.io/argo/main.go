// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package main

import (
	"fmt"
	"os"

	"applatix.io/argo/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
