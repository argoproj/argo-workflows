package validation

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"sigs.k8s.io/yaml"
)

func ValidateArgoYamlRecursively(fromPath string, skipFileNames []string) (map[string][]string, error) {
	schemaBytes, err := os.ReadFile("../api/jsonschema/schema.json")
	if err != nil {
		return nil, err
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		return nil, err
	}

	c := jsonschema.NewCompiler()
	// api/jsonschema/schema.json declares draft 2020-12, for which format
	// assertion is off by default; gojsonschema (the library this replaced)
	// always asserted format, so this restores that behaviour for the one
	// format either library recognizes, date-time.
	c.AssertFormat()
	if err := c.AddResource("schema.json", schemaDoc); err != nil {
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

		var doc any
		if err := yaml.Unmarshal(yamlBytes, &doc); err != nil {
			return err
		}

		docMap, _ := doc.(map[string]any)
		kind, _ := docMap["kind"].(string)
		schema, compileErr := c.Compile("schema.json#/definitions/io.argoproj.workflow.v1alpha1." + kind)
		if compileErr != nil {
			failed[path] = []string{fmt.Sprintf("unknown kind %q", kind)}
			return nil
		}

		if validationErr := schema.Validate(doc); validationErr != nil {
			ve, ok := validationErr.(*jsonschema.ValidationError)
			if !ok {
				return validationErr
			}
			if errs := realErrors(ve.DetailedOutput()); len(errs) > 0 {
				errorDescriptions := make([]string, 0, len(errs))
				for _, e := range errs {
					loc := e.InstanceLocation
					if loc == "" {
						loc = "(root)"
					}
					errorDescriptions = append(errorDescriptions, fmt.Sprintf("%s in %s", e.Error, loc))
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

// realErrors flattens a jsonschema.OutputUnit tree down to its field-level
// leaves. Compiling the document's own "kind" definition directly (rather
// than validating against the schema's root oneOf across all workflow
// kinds) means every remaining error already belongs to the document, so
// nothing needs filtering.
func realErrors(u *jsonschema.OutputUnit) []*jsonschema.OutputUnit {
	if u.Error != nil {
		return []*jsonschema.OutputUnit{u}
	}
	var errs []*jsonschema.OutputUnit
	for i := range u.Errors {
		errs = append(errs, realErrors(&u.Errors[i])...)
	}
	return errs
}
