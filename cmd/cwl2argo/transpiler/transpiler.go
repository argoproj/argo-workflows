package transpiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	CWLVersion = "v1.2"
)

func TranspileCommandlineTool(cl CommandlineTool, inputs map[string]interface{}, outputFile string) error {
	wf, err := EmitCommandlineTool(&cl, inputs)
	if err != nil {
		return err
	}
	// HACK: yaml Marshalling doesn't marshal correctly
	// therefore we turn the Workflow to map[string]interface and marshal that
	data, err := json.Marshal(wf)
	if err != nil {
		return err
	}

	m := make(map[string]interface{})
	err = json.Unmarshal(data, &m)
	if err != nil {
		return err
	}
	data, err = yaml.Marshal(m)
	return os.WriteFile(outputFile, data, 0644)
}

func TranspileFile(inputFile string, inputsFile string, outputFile string) error {

	log.Warn("Currently the transpiler expects preprocessed CWL input, sbpack is the recommended way of preprocessing, use cwlpack from here https://github.com/rabix/sbpack/tree/84bd7867a0630a826280a702db715377aa879f6a")
	var cwl map[string]interface{}
	var inputs map[string]interface{}

	def, err := os.ReadFile(inputFile)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(def, &cwl)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(inputsFile)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &inputs)
	if err != nil {
		return err
	}

	class, ok := cwl["class"]
	if !ok {
		return errors.New("<class> expected")
	}

	if class == "CommandLineTool" {
		var cliTool CommandlineTool
		err := yaml.Unmarshal(def, &cliTool)
		if err != nil {
			return err
		}

		return TranspileCommandlineTool(cliTool, inputs, outputFile)
	} else {
		return fmt.Errorf("%s is not supported as of yet", class)
	}

}
