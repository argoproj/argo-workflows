package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	schemaBytes, err := os.ReadFile("../api/jsonschema/schema.json")
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
		yamlBytes, err := os.ReadFile(filepath.Clean(path))
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

		incorrectError := false
		if !result.Valid() {
			errorDescriptions := []string{}
			for _, err := range result.Errors() {
				// port should be port number or port reference string, using string port number will cause issue
				// due swagger 2.0 limitation, we can only specify one data type (we use string, same as k8s api swagger).
				// Similarly, we cannot use string minAvailable either.
				if (strings.HasSuffix(err.Field(), "httpGet.port") || strings.HasSuffix(err.Field(), "podDisruptionBudget.minAvailable")) && err.Description() == "Invalid type. Expected: string, given: integer" {
					incorrectError = true
					continue
				} else {
					errorDescriptions = append(errorDescriptions, fmt.Sprintf("%s in %s", err.Description(), err.Context().String()))
				}
			}

			if !(incorrectError && len(errorDescriptions) == 1) {
				failed[path] = errorDescriptions
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return failed, nil
}
