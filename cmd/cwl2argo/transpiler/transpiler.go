package transpiler

import (
	"errors"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	CWLVersion = "v1.2"
)

func TranspileFile(inputFile string, inputsFile string, outputFile string) error {

	log.Warn("Currently the transpiler expects preprocessed CWL input, cwlpack is the recommended way of preprocessing, use cwlpack from here https://github.com/rabix/sbpack/tree/84bd7867a0630a826280a702db715377aa879f6a")
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

	return nil
}
