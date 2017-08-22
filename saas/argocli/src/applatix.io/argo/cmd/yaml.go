// Copyright 2015-2017 Applatix, Inc. All rights reserved.
package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"applatix.io/template"
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
	Short: "Validate a directory containing YAML files, or a list of multiple YAML files",
	Run:   validateYAML,
}

func isDir(filePath string) bool {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return fileInfo.IsDir()

}
func validateYAML(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.HelpFunc()(cmd, args)
		os.Exit(1)
	}
	validateDir := isDir(args[0])
	var ctx *template.TemplateBuildContext
	var err error
	if validateDir {
		if len(args) > 1 {
			fmt.Printf("Validation of a single directory supported")
			os.Exit(1)
		}
		fmt.Printf("Verifying all yaml files in directory: %s\n", args[0])
		ctx, err = buildContextFromDir(args[0], !yamlValidateArgs.failFast)
	} else {
		yamlFiles := make([]string, 0)
		for _, filePath := range args {
			if isDir(filePath) {
				fmt.Printf("Validate against a list of files or a single directory, not both")
				os.Exit(1)
			}
			yamlFiles = append(yamlFiles, filePath)
		}
		ctx, err = buildContextFromFiles(yamlFiles)
	}
	exitCode := 0
	if ctx != nil {
		if !ctx.IgnoreErrors {
			firstErr, filePath := ctx.FirstError()
			if firstErr != nil {
				fmt.Printf("[\033[1;30m%s\033[0m]\n", filePath)
				fmt.Printf(" - \033[31mERROR\033[0m %s: %v\n", firstErr.Template.GetName(), firstErr.AXErr)
				os.Exit(1)
			}
		}
		for filePath, templates := range ctx.PathToTemplate {
			printedFilePath := false
			for _, tmpl := range templates {
				templateName := tmpl.GetName()
				result, exists := ctx.Results[templateName]
				//fmt.Println(result, exists, err)
				if exists {
					if !printedFilePath {
						fmt.Printf("[\033[1;30m%s\033[0m]\n", filePath)
						printedFilePath = true
					}
					if result.AXErr != nil {
						exitCode = 1
						fmt.Printf(" - \033[31mERROR\033[0m %s: %v\n", templateName, result.AXErr)
					} else {
						fmt.Printf(" - \033[32mOK\033[0m    %s\n", templateName)
					}
				}
			}
		}
	} else {
		exitCode = 1
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
	os.Exit(exitCode)
}

func buildContextFromDir(dirName string, ignoreErrors bool) (*template.TemplateBuildContext, error) {
	ctx := template.NewTemplateBuildContext()
	ctx.Strict = yamlValidateArgs.strict
	ctx.IgnoreErrors = ignoreErrors
	axErr := ctx.ParseDirectory(dirName)
	if axErr != nil && !ctx.IgnoreErrors {
		return ctx, axErr
	}
	axErr = ctx.Validate()
	if axErr != nil && !ctx.IgnoreErrors {
		return ctx, axErr
	}
	return ctx, nil
}

func buildContextFromFiles(yamlFiles []string) (*template.TemplateBuildContext, error) {
	fmt.Printf("Verifying yaml files: %s\n", strings.Join(yamlFiles, ", "))
	context := template.NewTemplateBuildContext()
	context.Strict = yamlValidateArgs.strict
	context.IgnoreErrors = !yamlValidateArgs.failFast

	for _, path := range yamlFiles {
		body, err := ioutil.ReadFile(path)
		if err != nil && !context.IgnoreErrors {
			err := fmt.Errorf("Can't read from file: %s, err: %s\n", path, err)
			return nil, err
		}
		axErr := context.ParseFile(body, path)
		if axErr != nil && !context.IgnoreErrors {
			return context, axErr
		}
	}
	axErr := context.Validate()
	if axErr != nil && !context.IgnoreErrors {
		return context, axErr
	}
	return context, nil
}
