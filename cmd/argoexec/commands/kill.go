package commands

import (
	"fmt"
	"strconv"
	"syscall"

	"github.com/argoproj/argo-workflows/v3/workflow/executor/osspecific"

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
			fmt.Printf("killing %d with %v\n", pid, sig)
			if err := osspecific.Kill(pid, sig); err != nil {
				return err
			}
			return nil
		},
	}
}
