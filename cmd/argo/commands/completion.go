package commands

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"
)

const (
	bashCompletionFunc = `
__argo_get_workflow() {
	local argo_out
	if argo_out=$(argo list --output name 2>/dev/null); then
		COMPREPLY+=( $( compgen -W "${argo_out[*]}" -- "$cur" ) )
	fi
}

__argo_custom_func() {
	case ${last_command} in
		argo_delete | argo_get | argo_logs |\
		argo_resubmit | argo_resume | argo_retry | argo_suspend |\
		argo_terminate | argo_wait | argo_watch)
			__argo_get_workflow
			return
			;;
		*)
			;;
	esac
}
	`
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
			shell := args[0]
			rootCommand := NewCommand()
			rootCommand.BashCompletionFunction = bashCompletionFunc
			availableCompletions := map[string]func(io.Writer) error{
				"bash": rootCommand.GenBashCompletion,
				"zsh":  rootCommand.GenZshCompletion,
			}
			completion, ok := availableCompletions[shell]
			if !ok {
				fmt.Printf("Invalid shell '%s'. The supported shells are bash and zsh.\n", shell)
				os.Exit(1)
			}
			if err := completion(os.Stdout); err != nil {
				log.Fatal(err)
			}
		},
	}

	return command
}
