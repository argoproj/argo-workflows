package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"sigs.k8s.io/yaml"
)

func ValidateArgoYamlRecursively(fromPath string, skipFileNames []string) (map[string][]string, error) {
	schemaBytes, err := os.ReadFile("../api/jsonschema/schema.json")
	if err != nil {
		return nil, err
	}

	schema, err := jsonschema.CompileString("schema.json", string(schemaBytes))
	if err != nil {
		return nil, err
	}

	failed := map[string][]string{}

	err = filepath.Walk(fromPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if slices.Contains(skipFileNames, info.Name()) {
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

		var doc any
		if err := json.Unmarshal(jsonDoc, &doc); err != nil {
			return err
		}

		if validationErr := schema.Validate(doc); validationErr != nil {
			ve, ok := validationErr.(*jsonschema.ValidationError)
			if !ok {
				return validationErr
			}
			if errs := realErrors(ve); len(errs) > 0 {
				errorDescriptions := make([]string, 0, len(errs))
				for _, e := range errs {
					errorDescriptions = append(errorDescriptions, fmt.Sprintf("%s in %s", e.Message, e.InstanceLocation))
				}
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

// isAcceptedTypeMismatch reports whether err is the one known, accepted
// mismatch between the schema and the examples: httpGet.port and
// podDisruptionBudget.minAvailable are given as YAML integers even though
// the schema (mirroring the Kubernetes swagger) requires a string, because
// swagger 2.0 cannot express "string or integer" for a single field.
func isAcceptedTypeMismatch(err *jsonschema.ValidationError) bool {
	return (strings.HasSuffix(err.InstanceLocation, "/httpGet/port") || strings.HasSuffix(err.InstanceLocation, "/podDisruptionBudget/minAvailable")) &&
		err.Message == "expected string, but got number"
}

// realErrors flattens a jsonschema.ValidationError tree down to the
// field-level errors that aren't the accepted mismatch above. The schema's
// top-level "oneOf" (a Workflow/WorkflowTemplate/CronWorkflow/... union)
// means every document legitimately fails all but one branch, so a oneOf
// branch is treated as satisfied once the accepted mismatch is excluded,
// same as when the previous xeipuuv/gojsonschema-based validator ignored
// that single remaining error.
func realErrors(err *jsonschema.ValidationError) []*jsonschema.ValidationError {
	if len(err.Causes) == 0 {
		if isAcceptedTypeMismatch(err) {
			return nil
		}
		return []*jsonschema.ValidationError{err}
	}

	if strings.HasSuffix(err.KeywordLocation, "/oneOf") {
		for _, branch := range err.Causes {
			if len(realErrors(branch)) == 0 {
				return nil
			}
		}
	}

	var errs []*jsonschema.ValidationError
	for _, cause := range err.Causes {
		errs = append(errs, realErrors(cause)...)
	}
	return errs
}
