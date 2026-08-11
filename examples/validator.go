package validation

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"sigs.k8s.io/yaml"
)

func ValidateArgoYamlRecursively(fromPath string, skipFileNames []string) (map[string][]string, error) {
	schemaBytes, err := os.ReadFile("../api/jsonschema/schema.json")
	if err != nil {
		return nil, err
	}

	// Load and compile the schema once, then reuse it for every file.
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", schemaDoc); err != nil {
		return nil, err
	}
	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return nil, err
	}

	printer := message.NewPrinter(language.English)
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

		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonDoc))
		if err != nil {
			return err
		}

		validationErr := schema.Validate(doc)
		if validationErr == nil {
			return nil
		}
		var ve *jsonschema.ValidationError
		if !errors.As(validationErr, &ve) {
			return validationErr
		}

		residual := realValidationErrors(ve)
		if len(residual) > 0 {
			errorDescriptions := make([]string, 0, len(residual))
			for _, leaf := range residual {
				errorDescriptions = append(errorDescriptions,
					fmt.Sprintf("%s in /%s", leaf.ErrorKind.LocalizedString(printer), strings.Join(leaf.InstanceLocation, "/")))
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

// realValidationErrors reduces a ValidationError tree to the concrete leaf
// failures that actually matter, ignoring the known port/minAvailable quirk.
//
// The top-level schema is a oneOf/anyOf over every Argo resource type, so a
// valid manifest still produces errors for every branch it is *not* (e.g.
// "value must be 'CronWorkflow'"). For those combinator nodes we therefore
// attribute only the closest-matching branch (fewest residual errors) instead
// of every branch's noise.
func realValidationErrors(e *jsonschema.ValidationError) []*jsonschema.ValidationError {
	if len(e.Causes) == 0 {
		if isBenignStringTypeError(e) {
			return nil
		}
		return []*jsonschema.ValidationError{e}
	}

	switch e.ErrorKind.(type) {
	case *kind.OneOf, *kind.AnyOf:
		var best []*jsonschema.ValidationError
		for i, cause := range e.Causes {
			errs := realValidationErrors(cause)
			if i == 0 || len(errs) < len(best) {
				best = errs
			}
			if len(best) == 0 {
				break
			}
		}
		return best
	default:
		var all []*jsonschema.ValidationError
		for _, cause := range e.Causes {
			all = append(all, realValidationErrors(cause)...)
		}
		return all
	}
}

// isBenignStringTypeError reports whether e is the known "string expected but a
// number was given" error at httpGet.port / podDisruptionBudget.minAvailable.
// The schema can only declare one type (string, matching the k8s API swagger)
// due to a Swagger 2.0 limitation, so an integer value there is acceptable.
func isBenignStringTypeError(e *jsonschema.ValidationError) bool {
	typeErr, ok := e.ErrorKind.(*kind.Type)
	if !ok || !slices.Contains(typeErr.Want, "string") {
		return false
	}
	field := strings.Join(e.InstanceLocation, ".")
	return strings.HasSuffix(field, "httpGet.port") || strings.HasSuffix(field, "podDisruptionBudget.minAvailable")
}
