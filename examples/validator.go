package validation

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"sigs.k8s.io/yaml"
)

func ValidateArgoYamlRecursively(fromPath string, skipFileNames []string) (map[string][]string, error) {
	schemaBytes, err := os.ReadFile("../api/jsonschema/schema.json")
	if err != nil {
		return nil, err
	}

	c := jsonschema.NewCompiler()
	c.AssertFormat = true
	if err := c.AddResource("schema.json", bytes.NewReader(schemaBytes)); err != nil {
		return nil, err
	}
	schema, err := c.Compile("schema.json")
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

		var doc any
		if err := yaml.Unmarshal(yamlBytes, &doc); err != nil {
			return err
		}

		if validationErr := schema.Validate(doc); validationErr != nil {
			ve, ok := validationErr.(*jsonschema.ValidationError)
			if !ok {
				return validationErr
			}
			if errs := realErrors(doc, ve); len(errs) > 0 {
				errorDescriptions := make([]string, 0, len(errs))
				for _, e := range errs {
					loc := e.InstanceLocation
					if loc == "" {
						loc = "(root)"
					}
					errorDescriptions = append(errorDescriptions, fmt.Sprintf("%s in %s", e.Message, loc))
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
// swagger 2.0 cannot express "string or integer" for a single field. Only
// integral values are tolerated; a non-integral number at these locations is
// still a genuine mismatch.
func isAcceptedTypeMismatch(doc any, err *jsonschema.ValidationError) bool {
	if !((strings.HasSuffix(err.InstanceLocation, "/httpGet/port") || strings.HasSuffix(err.InstanceLocation, "/podDisruptionBudget/minAvailable")) &&
		err.Message == "expected string, but got number") {
		return false
	}
	val, ok := jsonPointerValue(doc, err.InstanceLocation)
	if !ok {
		return false
	}
	num, ok := val.(float64)
	return ok && num == math.Trunc(num)
}

// jsonPointerValue resolves the RFC 6901 JSON Pointer used by
// ValidationError.InstanceLocation against the decoded document.
func jsonPointerValue(doc any, pointer string) (any, bool) {
	cur := doc
	for _, tok := range strings.Split(pointer, "/") {
		if tok == "" {
			continue
		}
		tok = strings.NewReplacer("~1", "/", "~0", "~").Replace(tok)
		switch v := cur.(type) {
		case map[string]any:
			val, ok := v[tok]
			if !ok {
				return nil, false
			}
			cur = val
		case []any:
			idx, err := strconv.Atoi(tok)
			if err != nil || idx < 0 || idx >= len(v) {
				return nil, false
			}
			cur = v[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

// realErrors flattens a jsonschema.ValidationError tree down to the
// field-level errors that aren't the accepted mismatch above. The schema's
// top-level "oneOf" (a Workflow/WorkflowTemplate/CronWorkflow/... union)
// means every document legitimately fails all but one branch: when exactly
// one branch's "/kind" const matches the document's own kind, only that
// branch's errors are reported instead of every branch's, falling back to
// the full set when the matching branch can't be determined.
func realErrors(doc any, err *jsonschema.ValidationError) []*jsonschema.ValidationError {
	if len(err.Causes) == 0 {
		if isAcceptedTypeMismatch(doc, err) {
			return nil
		}
		return []*jsonschema.ValidationError{err}
	}

	if err.KeywordLocation == "/oneOf" {
		if branch := matchingKindBranch(err.Causes); branch != nil {
			return realErrors(doc, branch)
		}
	}

	var errs []*jsonschema.ValidationError
	for _, cause := range err.Causes {
		errs = append(errs, realErrors(doc, cause)...)
	}
	return errs
}

// matchingKindBranch returns the single oneOf branch whose "/kind" const
// error is absent (i.e. the branch matching the document's own kind), or
// nil if zero or more than one branch qualifies.
func matchingKindBranch(branches []*jsonschema.ValidationError) *jsonschema.ValidationError {
	var matched *jsonschema.ValidationError
	for _, branch := range branches {
		if hasKindMismatch(branch) {
			continue
		}
		if matched != nil {
			return nil
		}
		matched = branch
	}
	return matched
}

// hasKindMismatch reports whether the branch's error tree contains an error
// for the document's "/kind" field, i.e. the document's kind doesn't match
// this branch's expected kind.
func hasKindMismatch(err *jsonschema.ValidationError) bool {
	if err.InstanceLocation == "/kind" {
		return true
	}
	for _, cause := range err.Causes {
		if hasKindMismatch(cause) {
			return true
		}
	}
	return false
}
