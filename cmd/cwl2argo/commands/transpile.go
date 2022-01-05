package commands

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v3/cmd/cwl2argo/transpiler"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func lengthError() error {
	return errors.New("length of filename should always be greater than the length of the extension")
}

func invalidFileError(inputFileExt string) error {
	return fmt.Errorf("invalid file extension %s, only common workflow language (.cwl) files are allowed", inputFileExt)
}
func extractNoExtFileName(filename string, ext string) (string, error) {
	if len(filename) <= len(ext) {
		return "", lengthError()
	}
	name := filename[0 : len(filename)-len(ext)]
	return name, nil
}

func processFile(inputFile string, inputsFile string) {

	ext := filepath.Ext(inputFile)
	if ext != ".cwl" {
		log.Fatalf("%+v", invalidFileError(ext))
	}
	name, err := extractNoExtFileName(inputFile, ext)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	newName := fmt.Sprintf("argo_%s.yaml", name)
	log.Infof("Transpiling file %s with extension %s and ext free name %s to %s", inputFile, ext, name, newName)
	err = transpiler.TranspileFile(inputFile, inputsFile, newName)
	if err != nil {
		log.Fatalf("%+v", err)
	}

}

// Cobra command to transpile
func NewTranspileCommand() *cobra.Command {

	command := cobra.Command{
		Use:   "transpile",
		Short: "input common workflow language file",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.New("transpile only accepts two arguments <WORKFLOW.cwl> and <INPUTS.(yml|json|cwl)>")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			processFile(args[0], args[1])
		},
	}

	return &command
}
