package transpiler

import (
	"encoding/json"
	"errors"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	CWLVersion = "v1.0"
)

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

	if _, ok := cwl["class"]; !ok {
		return errors.New("<class> expected")
	}

	switch cwl["class"] {
	case "CommandLineTool":
		log.Info("Transpiling CommandLineTool")
		tool, err := FillCommandlineTool(cwl)
		_ = tool
		if err != nil {
			return err
		}

	case "Workflow":
		return errors.New("Workflows have not been implemented yet")

	default:
		return errors.New("")
	}

	cliTool, err := FillCommandlineTool(cwl)
	if err != nil {
		return err
	}
	err = TypeCheckCommandlineTool(cliTool, inputs)

	wf, err := EmitArgo(cliTool)
	if err != nil {
		return err
	}
	by, err := json.Marshal(&wf)
	if err != nil {
		return err
	}
	dynYaml := make(map[string]interface{})
	err = yaml.Unmarshal(by, dynYaml)
	if err != nil {
		return err
	}
	by, err = yaml.Marshal(dynYaml)
	if err != nil {
		return err
	}

	return nil
}
