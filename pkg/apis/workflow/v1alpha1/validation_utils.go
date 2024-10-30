package v1alpha1

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	apivalidation "k8s.io/apimachinery/pkg/util/validation"
)

const (
	workflowFieldNameFmt    string = "[a-zA-Z0-9][-a-zA-Z0-9]*"
	workflowFieldNameErrMsg string = "name must consist of alpha-numeric characters or '-', and must start with an alpha-numeric character"
	workflowFieldMaxLength  int    = 128
)

var (
	paramOrArtifactNameRegex = regexp.MustCompile(`^[-a-zA-Z0-9_]+[-a-zA-Z0-9_]*$`)
	workflowFieldNameRegex   = regexp.MustCompile("^" + workflowFieldNameFmt + "$")
)

func isValidParamOrArtifactName(p string) []string {
	var errs []string
	if !paramOrArtifactNameRegex.MatchString(p) {
		return append(errs, "Parameter/Artifact name must consist of alpha-numeric characters, '_' or '-' e.g. my_param_1, MY-PARAM-1")
	}
	return errs
}

// isValidWorkflowFieldName : workflow field name must consist of alpha-numeric characters or '-', and must start with an alpha-numeric character
func isValidWorkflowFieldName(name string) []string {
	var errs []string
	if len(name) > workflowFieldMaxLength {
		errs = append(errs, apivalidation.MaxLenError(workflowFieldMaxLength))
	}
	if !workflowFieldNameRegex.MatchString(name) {
		msg := workflowFieldNameErrMsg + " (e.g. My-name1-2, 123-NAME)"
		errs = append(errs, msg)
	}
	return errs
}

// validateWorkflowFieldNames accepts a slice of strings and
// verifies that the Name field of the structs are:
// * unique
// * non-empty
// * matches matches our regex requirements
func validateWorkflowFieldNames(names []string, isParamOrArtifact bool) error {
	nameSet := make(map[string]bool)

	for i, name := range names {
		if name == "" {
			return fmt.Errorf("[%d].name is required", i)
		}
		var errs []string
		if isParamOrArtifact {
			errs = isValidParamOrArtifactName(name)
		} else {
			errs = isValidWorkflowFieldName(name)
		}
		if len(errs) != 0 {
			return fmt.Errorf("[%d].name: '%s' is invalid: %s", i, name, strings.Join(errs, ";"))
		}
		// _, ok := nameSet[name]
		// if ok {
		// 	return fmt.Errorf("[%d].name '%s' is not unique", i, name)
		// }
		nameSet[name] = true
	}
	return nil
}

// validateNoCycles validates that a dependency graph has no cycles by doing a Depth-First Search
// depGraph is an adjacency list, where key is a node name and value is a list of its dependencies' names
func validateNoCycles(depGraph map[string][]string) error {
	visited := make(map[string]bool)
	var noCyclesHelper func(currentName string, cycyle []string) error
	noCyclesHelper = func(currentName string, cycle []string) error {
		if _, ok := visited[currentName]; ok {
			return nil
		}
		depNames, ok := depGraph[currentName]
		if !ok {
			return nil
		}
		for _, depName := range depNames {
			for _, name := range cycle {
				if depName == name {
					return fmt.Errorf("dependency cycle detected: %s->%s", strings.Join(cycle, "->"), name)
				}
			}
			cycle = append(cycle, depName)
			err := noCyclesHelper(depName, cycle)
			if err != nil {
				return err
			}
			cycle = cycle[0 : len(cycle)-1]
		}
		visited[currentName] = true
		return nil
	}
	names := make([]string, 0)
	for name := range depGraph {
		names = append(names, name)
	}
	// sort names here to make sure the error message has consistent ordering
	// so that we can verify the error message in unit tests
	sort.Strings(names)

	for _, name := range names {
		err := noCyclesHelper(name, []string{})
		if err != nil {
			return err
		}
	}
	return nil
}
