package commands

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"

	gops "github.com/mitchellh/go-ps"
	"github.com/spf13/cobra"
)

func NewKillCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "kill SIGNAL",
		Long:         "signal every root process",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("expected 1 arg")
			}
			sig, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			ps, err := gops.Processes()
			if err != nil {
				return err
			}
			for _, p := range ps {
				if p.PPid() > 0 {
					continue
				}
				pid := p.Pid()
				fmt.Printf("signaling %d with %d\n", pid, sig)
				p, err := os.FindProcess(pid)
				if err != nil {
					return err
				}
				if err := p.Signal(syscall.Signal(sig)); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
