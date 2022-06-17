package commands

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

func NewKillCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "kill SIGNAL PID",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			signum, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			pid, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}
			sig := syscall.Signal(signum)
			p, err := os.FindProcess(pid)
			if err != nil {
				return err
			}
			fmt.Printf("killing %d with %v\n", pid, sig)
			if err := p.Signal(sig); err != nil {
				return err
			}
			return nil
		},
	}
}
