package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gobuffalo/packr"
	"github.com/gobuffalo/packr/builder"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:                "install",
	Short:              "Wraps the go install command with packr",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			input = args[len(args)-1]
			if !strings.HasPrefix(input, ".") {
				input = filepath.Join(packr.GoPath(), "src", input)
				if _, err := os.Stat(input); err != nil {
					return errors.WithStack(err)
				}
			}
		}
		defer builder.Clean(input)
		b := builder.New(context.Background(), input)
		err := b.Run()
		if err != nil {
			return errors.WithStack(err)
		}

		cargs := []string{"install"}
		cargs = append(cargs, args...)
		cp := exec.Command(packr.GoBin(), cargs...)
		cp.Stderr = os.Stderr
		cp.Stdin = os.Stdin
		cp.Stdout = os.Stdout

		return cp.Run()
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
