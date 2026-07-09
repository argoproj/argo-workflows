package dag

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Substitutor defines an interface for substituting variables in a string.
type Substitutor interface {
	// Substitute substitutes variables in the given template string.
	Substitute(template string, scope map[string]string) (string, error)
}

// Task represents a task in a workflow (DAG or Steps) that has a name and dependency information.
type Task interface {
	GetName() string
	GetDisplayName() string
	GetDepends() string
	GetDependencies() []string
	GetContinueOn() *wfv1.ContinueOn
	GetTemplateReferenceHolder() wfv1.TemplateReferenceHolder
	GetArguments() wfv1.Arguments
	GetWithItems() []wfv1.Item
	GetWithParam() string
	GetWithSequence() *wfv1.Sequence
	GetWhen() string
	GetHooks() wfv1.LifecycleHooks
	GetExitHook(args wfv1.Arguments) *wfv1.LifecycleHook
	ContinuesOn(phase wfv1.NodePhase) bool
	Expand(ctx context.Context, scope map[string]string, substitutor Substitutor) ([]Task, error)
}

// HasExpansion reports whether the task uses withItems/withParam/withSequence
// and therefore expands into a TaskGroup of per-item children.
func HasExpansion(t Task) bool {
	return t.GetWithItems() != nil || t.GetWithParam() != "" || t.GetWithSequence() != nil
}

// DAGTask adapts wfv1.DAGTask to the Task interface.
//
//nolint:revive // name mirrors the wrapped wfv1.DAGTask; the unprefixed Task is the interface it implements
type DAGTask struct {
	*wfv1.DAGTask
}

func (t *DAGTask) Expand(ctx context.Context, scope map[string]string, substitutor Substitutor) ([]Task, error) {
	expanded, err := ExpandTask(ctx, *t.DAGTask, scope, substitutor)
	if err != nil {
		return nil, err
	}
	tasks := make([]Task, len(expanded))
	for i := range expanded {
		tasks[i] = &DAGTask{DAGTask: &expanded[i]}
	}
	return tasks, nil
}

func (t *DAGTask) GetName() string {
	return t.Name
}

func (t *DAGTask) GetDisplayName() string {
	return t.Name
}

func (t *DAGTask) GetDepends() string {
	return t.Depends
}

func (t *DAGTask) GetDependencies() []string {
	return t.Dependencies
}

func (t *DAGTask) GetContinueOn() *wfv1.ContinueOn {
	return t.ContinueOn
}

func (t *DAGTask) GetTemplateReferenceHolder() wfv1.TemplateReferenceHolder {
	return t
}

func (t *DAGTask) GetArguments() wfv1.Arguments {
	return t.Arguments
}

func (t *DAGTask) GetWithItems() []wfv1.Item {
	return t.WithItems
}

func (t *DAGTask) GetWithParam() string {
	return t.WithParam
}

func (t *DAGTask) GetWithSequence() *wfv1.Sequence {
	return t.WithSequence
}

func (t *DAGTask) GetWhen() string {
	return t.When
}

func (t *DAGTask) GetHooks() wfv1.LifecycleHooks {
	return t.Hooks
}
