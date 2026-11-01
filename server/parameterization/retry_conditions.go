package parameterization

import (
	"fmt"
	"slices"
	"sort"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// Conditions consume inputs before dispatch, including outputs of transitive
// ancestors. Add those data edges to the analysis without changing the native
// Workflow graph. Native Argo evaluates the condition after retry.
func retryConditionDependencies(task *wfv1.DAGTask, tasks map[string]wfv1.DAGTask) (p.ParameterReferences, error) {
	task.Dependencies = slices.Clone(task.Dependencies)
	refs, err := p.References(task.When)
	if err != nil {
		return nil, err
	}
	ancestors := map[string]bool{}
	var visit func(string)
	visit = func(name string) {
		if ancestors[name] {
			return
		}
		ancestors[name] = true
		for _, dep := range tasks[name].Dependencies {
			visit(dep)
		}
	}
	for _, dep := range task.Dependencies {
		visit(dep)
	}
	for _, scope := range []string{"tasks", "steps"} {
		for producer := range refs[scope] {
			if producer == "*" || producer == task.Name || !ancestors[producer] {
				return nil, fmt.Errorf("condition at %s requires statically named upstream dependencies", task.Name)
			}
			if !slices.Contains(task.Dependencies, producer) {
				task.Dependencies = append(slices.Clone(task.Dependencies), producer)
			}
		}
	}
	sort.Strings(task.Dependencies)
	return refs, nil
}
