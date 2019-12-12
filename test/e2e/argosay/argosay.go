package main

import (
	"os"
	"time"

	"github.com/spf13/cobra"
)

func main() {
	var exitCode int
	var sleep time.Duration

	command := cobra.Command{
		Use: "argosay",
		Run: func(cmd *cobra.Command, args []string) {
			time.Sleep(sleep)
			os.Exit(exitCode)
		},
	}

	command.Flags().IntVar(&exitCode, "exit-code", 0, "Exit code")
	command.Flags().DurationVar(&sleep, "sleep", 0, "Sleep")

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
