package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewCompletionCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "completion SHELL",
		Short: "output shell completion code for the specified shell (bash or zsh)",
		Long: `Write bash or zsh shell completion code to standard output.

For bash, ensure you have bash completions installed and enabled.
To access completions in your current shell, run
$ source <(argo completion bash)
Alternatively, write it to a file and source in .bash_profile

For zsh, output to a file in a directory referenced by the $fpath shell
variable.
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			var shell = args[0]
			rootCommand := NewCommand()
			if shell == "bash" {
				rootCommand.GenBashCompletion(os.Stdout)
			} else if shell == "zsh" {
				rootCommand.GenZshCompletion(os.Stdout)
			} else {
				fmt.Printf("Invalid shell '%s'. The supported shells are bash and zsh.\n", shell)
				os.Exit(1)
			}
		},
	}

	return command
}
