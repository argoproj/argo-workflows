package commands

import (
	"os"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

func NewKillCommand() *cobra.Command {
	var signal int
	command := cobra.Command{
		Use: "kill",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			p, err := os.FindProcess(pid)
			if err != nil {
				return err
			}
			return p.Signal(syscall.Signal(signal))
		},
	}
	command.Flags().IntVar(&signal, "signal", 0, "signal")
	return &command
}
