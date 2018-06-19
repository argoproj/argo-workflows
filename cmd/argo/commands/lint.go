package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/argoproj/argo/errors"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewLintCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:   "lint (DIRECTORY | FILE1 FILE2 FILE3...)",
		Short: "validate a directory or specific workflow YAML files",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			validateDir := cmdutil.MustIsDir(args[0])
			var err error
			if validateDir {
				if len(args) > 1 {
					fmt.Printf("Validation of a single directory supported")
					os.Exit(1)
				}
				fmt.Printf("Verifying all yaml files in directory: %s\n", args[0])
				err = lintYAMLDir(args[0])
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
					err = lintYAMLFile(yamlFile)
					if err != nil {
						break
					}
				}
			}
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("YAML validated\n")
		},
	}
	return command
}

func lintYAMLDir(dirPath string) error {
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		fileExt := filepath.Ext(info.Name())
		if fileExt != ".yaml" && fileExt != ".yml" {
			return nil
		}
		return lintYAMLFile(path)
	}
	return filepath.Walk(dirPath, walkFunc)
}

// lintYAMLFile lints multiple workflow manifest in a single yaml file. Ignores non-workflow manifests
func lintYAMLFile(filePath string) error {
	body, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "Can't read from file: %s, err: %v", filePath, err)
	}
	workflows, err := splitYAMLFile(body)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "%s failed to parse: %v", filePath, err)
	}
	for _, wf := range workflows {
		err = common.ValidateWorkflow(&wf, true)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "%s: %s", filePath, err.Error())
		}
	}
	return nil
}
