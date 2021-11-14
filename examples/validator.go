package validation

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
	"sigs.k8s.io/yaml"
)

// https://stackoverflow.com/questions/10485743/contains-method-for-a-slice
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func ValidateArgoYamlRecursively(fromPath string, skipFileNames []string) (map[string][]string, error) {
	schemaBytes, err := ioutil.ReadFile("../api/jsonschema/schema.json")
	if err != nil {
		return nil, err
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaBytes))

	failed := map[string][]string{}

	err = filepath.Walk(fromPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if contains(skipFileNames, info.Name()) {
			// fmt.Printf("skipping %+v \n", info.Name())
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yaml" {
			return nil
		}
		yamlBytes, err := ioutil.ReadFile(filepath.Clean(path))
		if err != nil {
			return err
		}

		jsonDoc, err := yaml.YAMLToJSON(yamlBytes)
		if err != nil {
			return err
		}

		documentLoader := gojsonschema.NewStringLoader(string(jsonDoc))

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			return err
		}

		if !result.Valid() {
			errorDescriptions := []string{}
			for _, err := range result.Errors() {
				errorDescriptions = append(errorDescriptions, fmt.Sprintf("%s in %s", err.Description(), err.Context().String()))
			}
			failed[path] = errorDescriptions
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return failed, nil
}
