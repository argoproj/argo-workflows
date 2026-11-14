package parameterization

import (
	"fmt"
	"reflect"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// A reviewed retry may replace only implementations which no native invocation
// has visited. In particular, changing a shared template changes every call:
// one completed invocation is enough to freeze that template. The native graph,
// executor, credentials and completed approval boundaries remain unchanged.
func validateUnvisitedTemplates(wf *wfv1.Workflow, old, next *wfv1.WorkflowSpec) error {
	a, b := old.DeepCopy(), next.DeepCopy()
	a.Templates, b.Templates = nil, nil
	if !reflect.DeepEqual(a, b) {
		return fmt.Errorf("partial retry cannot change the visited graph, execution scope or credentials")
	}
	before, after := map[string]wfv1.Template{}, map[string]wfv1.Template{}
	for _, t := range old.Templates {
		before[t.Name] = t
	}
	for _, t := range next.Templates {
		after[t.Name] = t
	}
	visited := map[string]bool{old.Entrypoint: true}
	for _, node := range wf.Status.Nodes {
		if node.Type == wfv1.NodeTypeSkipped && node.Phase == wfv1.NodeOmitted {
			continue
		}
		if node.TemplateName != "" {
			visited[node.TemplateName] = true
		}
	}
	for name := range visited {
		a, ok := before[name]
		b, exists := after[name]
		if !ok || !exists || !reflect.DeepEqual(a, b) {
			return fmt.Errorf("partial retry cannot replace visited template %s", name)
		}
	}
	return nil
}
