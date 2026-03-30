package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

const (
	bashCompletionFunc = `
__argo_get_workflow() {
	local status="$1"
	local -a argo_out
	if argo_out=($(argo list --status="$status" --output name 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${argo_out[*]}" -- "$cur" ) )
	fi
}

__argo_get_workflow_template() {
	local -a argo_out
	if argo_out=($(argo template list --output name 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${argo_out[*]}" -- "$cur" ) )
	fi
}

__argo_get_cluster_workflow_template() {
	local -a argo_out
	if argo_out=($(argo cluster-template list --output name 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${argo_out[*]}" -- "$cur" ) )
	fi
}

__argo_get_cron_workflow() {
	local -a argo_out
	if argo_out=($(argo cron list --output name 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${argo_out[*]}" -- "$cur" ) )
	fi
}

__argo_get_logs() {
	# Determine if were completing a workflow or not.
	if [[ $prev == "logs" ]]; then
		__argo_get_workflow && return $?
 	fi
    local workflow=$prev
	# Otherwise, complete the list of pods
	local -a kubectl_out
	if kubectl_out=($(kubectl get pods --no-headers --selector=workflows.argoproj.io/workflow="${workflow}" 2>/dev/null | awk '{print $1}' 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${kubectl_out[*]}" -- "$cur" ) )
	fi
}

__argo_list_files() {
	COMPREPLY+=( $( compgen -f -o plusdirs -X '!*.@(yaml|yml|json)' -- "$cur" ) )
}

__argo_custom_func() {
	case ${last_command} in
		argo_delete | argo_get | argo_resubmit)
			__argo_get_workflow
			return
			;;
		argo_suspend | argo_terminate | argo_wait | argo_watch)
			__argo_get_workflow "Running,Pending"
			return
			;;
		argo_resume)
			__argo_get_workflow "Running"
			return
			;;
		argo_retry)
			__argo_get_workflow "Failed"
			return
			;;
		argo_logs)
			__argo_get_logs
			return
			;;
		argo_submit | argo_lint)
			__argo_list_files
			return
			;;
		argo_template_get | argo_template_delete)
			__argo_get_workflow_template
			return
			;;
		argo_template_create | argo_template_lint)
		    __argo_list_files
			return
			;;
		argo_cluster-template_get | argo_cluster-template_delete)
			__argo_get_cluster_workflow_template
			return
			;;
		argo_cluster-template_create | argo_cluster-template_lint)
		    __argo_list_files
			return
			;;
		argo_cron_get | argo_cron_delete | argo_cron_resume | argo_cron_suspend)
			__argo_get_cron_workflow
			return
			;;
		argo_cron_create | argo_cron_lint)
		    __argo_list_files
			return
			;;
		*)
			;;
	esac
}
	`
)

func NewCompletionCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "completion SHELL",
		Short: "output shell completion code for the specified shell (bash, zsh or fish)",
		Long: `Write bash, zsh or fish shell completion code to standard output.

For bash, ensure you have bash completions installed and enabled.
To access completions in your current shell, run
$ source <(argo completion bash)
Alternatively, write it to a file and source in .bash_profile

For zsh, output to a file in a directory referenced by the $fpath shell
variable.

For fish, output to a file in ~/.config/fish/completions
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			rootCommand := NewCommand()
			rootCommand.BashCompletionFunction = bashCompletionFunc
			availableCompletions := map[string]func(out io.Writer, cmd *cobra.Command) error{
				"bash": runCompletionBash,
				"zsh":  runCompletionZsh,
				"fish": runCompletionFish,
			}
			completion, ok := availableCompletions[shell]
			if !ok {
				return fmt.Errorf("invalid shell %q: supported shells are bash, zsh, and fish", shell)
			}
			return completion(os.Stdout, rootCommand)
		},
	}
	return command
}

func runCompletionBash(out io.Writer, cmd *cobra.Command) error {
	return cmd.GenBashCompletion(out)
}

func runCompletionZsh(out io.Writer, cmd *cobra.Command) error {
	return cmd.GenZshCompletion(out)
}

func runCompletionFish(out io.Writer, cmd *cobra.Command) error {
	return cmd.GenFishCompletion(out, true)
}
