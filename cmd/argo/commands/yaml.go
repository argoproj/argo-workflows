package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

// CLI options
var (
	yamlValidateArgs yamlValidateFlags
)

type yamlValidateFlags struct {
	failFast bool
	strict   bool
}

func init() {
	RootCmd.AddCommand(yamlCmd)
	yamlCmd.AddCommand(yamlValidateCmd)
	yamlValidateCmd.Flags().BoolVar(&yamlValidateArgs.failFast, "failfast", true, "Stop upon first validation error")
	yamlValidateCmd.Flags().BoolVar(&yamlValidateArgs.strict, "strict", true, "Do not accept unknown keys during validation")
}

var yamlCmd = &cobra.Command{
	Use:   "yaml",
	Short: "YAML commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

var yamlValidateCmd = &cobra.Command{
	Use:   "validate (DIRECTORY | FILE1 FILE2 FILE3...)",
	Short: "Validate a directory containing Workflow YAML files, or a list of multiple Workflow YAML files",
	Run:   validateYAML,
}

func validateYAML(cmd *cobra.Command, args []string) {
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
		err = validateYAMLDir(args[0])
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
			err = vaildateYAMLFile(yamlFile)
			if err != nil {
				break
			}
		}
	}
	if err != nil {
		fmt.Printf("YAML validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("YAML validated\n")
	os.Exit(0)
}

func validateYAMLDir(dirPath string) error {
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		fileExt := filepath.Ext(info.Name())
		if fileExt != ".yaml" && fileExt != ".yml" {
			return nil
		}
		return vaildateYAMLFile(path)
	}
	return filepath.Walk(dirPath, walkFunc)
}

func vaildateYAMLFile(filePath string) error {
	body, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "Can't read from file: %s, err: %v\n", filePath, err)
	}
	var wf wfv1.Workflow
	err = yaml.Unmarshal(body, &wf)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "Failed to parse %s: %v", filePath, err)
	}
	return nil
}
