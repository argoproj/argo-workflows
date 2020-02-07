package cron

import (
	"fmt"
	"os"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/validate"
)

func NewLintCommand() *cobra.Command {
	var (
		strict bool
	)
	var command = &cobra.Command{
		Use:   "lint (DIRECTORY | FILE1 FILE2 FILE3...)",
		Short: "validate a file or directory of cron workflow manifests",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			wftmplGetter := &LazyWorkflowTemplateGetter{}
			validateDir := cmdutil.MustIsDir(args[0])
			if validateDir {
				if len(args) > 1 {
					fmt.Printf("Validation of a single directory supported")
					os.Exit(1)
				}
				fmt.Printf("Verifying all cron workflow manifests in directory: %s\n", args[0])
				err := validate.LintCronWorkflowDir(wftmplGetter, args[0], strict)
				errors.CheckError(err)
			} else {
				yamlFiles := make([]string, 0)
				for _, filePath := range args {
					if cmdutil.MustIsDir(filePath) {
						fmt.Printf("Validate against a list of files or a single directory, not both")
						os.Exit(1)
					}
					yamlFiles = append(yamlFiles, filePath)
				}
				for _, yamlFile := range yamlFiles {
					err := validate.LintCronWorkflowFile(wftmplGetter, yamlFile, strict)
					errors.CheckError(err)
				}
			}
			fmt.Printf("CronWorkflow manifests validated\n")
		},
	}
	command.Flags().BoolVar(&strict, "strict", true, "perform strict workflow validation")
	return command
}
