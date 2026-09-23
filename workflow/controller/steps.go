package controller

import (
	"context"
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common/dag"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

// StepAdapter is an adapter for wfv1.WorkflowStep to implement the Task interface.
type StepAdapter struct {
	step         *wfv1.WorkflowStep
	dependencies []string
	groupIndex   int
}

func (s *StepAdapter) GetName() string {
	return stepTaskNameFor(s.groupIndex, s.step.Name)
}

// stepTaskNameFor is the task name of a step within a Steps template:
// "[<group index>].<step name>". The group index is what orders the groups;
// stepGroupIndexOf recovers it.
func stepTaskNameFor(groupIndex int, stepName string) string {
	return fmt.Sprintf("[%d].%s", groupIndex, stepName)
}

// stepGroupNodeName is the node name of the groupIndex-th step group of the
// Steps node stepsNodeName: "<steps node>[<group index>]". A step's node name
// is this followed by ".<step name>" (dag.TaskNodeName over stepTaskNameFor).
func stepGroupNodeName(stepsNodeName string, groupIndex int) string {
	return fmt.Sprintf("%s[%d]", stepsNodeName, groupIndex)
}

// stepGroupIndexOf recovers the step group index from a Steps task name
// produced by stepTaskNameFor. ok is false for any other name.
func stepGroupIndexOf(taskName string) (groupIndex int, ok bool) {
	n, _ := fmt.Sscanf(taskName, "[%d].", &groupIndex)
	return groupIndex, n == 1
}

func (s *StepAdapter) GetDisplayName() string {
	return s.step.Name
}

func (s *StepAdapter) GetTemplateReferenceHolder() wfv1.TemplateReferenceHolder {
	return s.step
}

func (s *StepAdapter) GetArguments() wfv1.Arguments {
	return s.step.Arguments
}

func (s *StepAdapter) GetWithItems() []wfv1.Item {
	return s.step.WithItems
}

func (s *StepAdapter) GetWithParam() string {
	return s.step.WithParam
}

func (s *StepAdapter) GetWithSequence() *wfv1.Sequence {
	return s.step.WithSequence
}

func (s *StepAdapter) GetWhen() string {
	return s.step.When
}

func (s *StepAdapter) GetDepends() string {
	return ""
}

func (s *StepAdapter) GetDependencies() []string {
	return s.dependencies
}

func (s *StepAdapter) GetContinueOn() *wfv1.ContinueOn {
	return s.step.ContinueOn
}

func (s *StepAdapter) ContinuesOn(phase wfv1.NodePhase) bool {
	return s.step.ContinuesOn(phase)
}

func (s *StepAdapter) GetHooks() wfv1.LifecycleHooks {
	return s.step.Hooks
}

func (s *StepAdapter) GetExitHook(args wfv1.Arguments) *wfv1.LifecycleHook {
	return s.step.GetExitHook(args)
}

func (s *StepAdapter) Expand(ctx context.Context, scope map[string]string, substitutor dag.Substitutor) ([]dag.Task, error) {
	// Construct a temporary DAGTask to reuse the DAG expansion logic. Every
	// WorkflowStep field that DAGTask also has must be carried over: the
	// expanded task is what the reconciler resolves the template from.
	dt := &dag.DAGTask{DAGTask: &wfv1.DAGTask{
		Name:         s.GetName(),
		Template:     s.step.Template,
		Inline:       s.step.Inline,
		Arguments:    s.step.Arguments,
		WithItems:    s.step.WithItems,
		WithParam:    s.step.WithParam,
		WithSequence: s.step.WithSequence,
		When:         s.step.When,
		ContinueOn:   s.step.ContinueOn,
		OnExit:       s.step.OnExit, //nolint:staticcheck // OnExit is deprecated but still honored for backward compatibility
		TemplateRef:  s.step.TemplateRef,
		Hooks:        s.step.Hooks,
		Dependencies: s.dependencies,
	}}
	expanded, err := dt.Expand(ctx, scope, substitutor)
	if err != nil {
		return nil, err
	}
	for i := range expanded {
		expanded[i] = expandedStepTask{Task: expanded[i]}
	}
	return expanded, nil
}

// expandedStepTask is one item of an expanded step. Its task name keeps the
// "[i]." group prefix the Engine schedules by; its display name, which is
// what {{steps.name}} and the steps.<name> scope keys carry, is the item name
// alone (e.g. "A(0:x)"), as it was before the Engine.
type expandedStepTask struct {
	dag.Task
}

func (t expandedStepTask) GetDisplayName() string {
	return stepNameOf(t.GetName())
}

// stepNameOf is the inverse of stepTaskNameFor: the step (or expanded item)
// name without the "[i]." group prefix.
func stepNameOf(taskName string) string {
	if _, name, ok := strings.Cut(taskName, "]."); ok && strings.HasPrefix(taskName, "[") {
		return name
	}
	return taskName
}

// executeSteps executes a Steps template by converting step groups into DAG tasks
// and delegating to the Engine for scheduling and reconciliation.
// The engine's evaluate-then-converge loop handles cascading instant completions
// (e.g. when-skipped, cache hits) within a single Execute call, so step groups
// that complete instantly are processed without extra reconcile cycles.
func (woc *wfOperationCtx) executeSteps(ctx context.Context, nodeName string, tmplCtx *templateresolution.TemplateContext, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		_, node = woc.initializeExecutableNode(ctx, nodeName, wfv1.NodeTypeSteps, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodeRunning, opts.nodeFlag, true)
	}

	defer func() {
		nodePhase, phaseErr := woc.wf.Status.Nodes.GetPhase(node.ID)
		if phaseErr != nil {
			woc.log.WithField("nodeID", node.ID).WithFatal().Error(ctx, "was unable to obtain nodePhase for nodeID")
			panic(fmt.Sprintf("unable to obtain nodePhase for %s", node.ID))
		}
		if nodePhase.Fulfilled(node.TaskResultSynced) {
			woc.killDaemonedChildren(ctx, node.ID)
		}
	}()

	var tasks []dag.Task
	var prevStepNames []string
	for i, stepGroup := range tmpl.Steps {
		// Create StepGroup node. Only [0] is linked to the Steps root here; [i>0]
		// is wired after engine.Execute once the previous group's children exist
		// (see linkStepGroups below).
		sgNodeName := stepGroupNodeName(nodeName, i)
		if _, err := woc.wf.GetNodeByName(sgNodeName); err != nil {
			_, _ = woc.initializeNode(ctx, sgNodeName, wfv1.NodeTypeStepGroup, tmplCtx.GetTemplateScope(), &wfv1.WorkflowStep{}, node.ID, wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
			if i == 0 {
				woc.addChildNode(ctx, nodeName, sgNodeName)
			}
		}

		var currentStepNames []string
		for _, step := range stepGroup.Steps {
			task := &StepAdapter{
				step:         &step,
				dependencies: prevStepNames,
				groupIndex:   i,
			}
			tasks = append(tasks, task)
			currentStepNames = append(currentStepNames, task.GetName())
		}
		prevStepNames = currentStepNames
	}

	engine := NewEngine(woc, nodeName, tmplCtx, tmpl, orgTmpl, node.ID, opts.onExitTemplate)
	engine.Execute(ctx, tasks)

	if err := woc.linkStepGroups(ctx, nodeName, tmpl); err != nil {
		return nil, err
	}
	return woc.wf.GetNodeByName(nodeName)
}

// linkStepGroups wires each StepGroup [i>0] as a child of the outbound nodes of
// every child of [i-1], mirroring legacy Steps graph semantics. If [i-1] has no
// children yet (e.g. empty withParam expansion), [i] is linked directly under
// [i-1]. addChildNode dedupes, so this is safe to call on every operate cycle.
//
// The linking is gated on [i-1] being fulfilled. Linking earlier would inject
// [i] into the descendant chain of an in-flight node — when childrenFulfilled()
// later recurses through that chain (e.g. during retry finalization), it would
// see [i]'s subtree as unfulfilled and skip synchronization lock release /
// retry completion.
func (woc *wfOperationCtx) linkStepGroups(ctx context.Context, nodeName string, tmpl *wfv1.Template) error {
	for i := 1; i < len(tmpl.Steps); i++ {
		sgNodeName := stepGroupNodeName(nodeName, i)
		prevSgNodeName := stepGroupNodeName(nodeName, i-1)
		prevSgNode, err := woc.wf.GetNodeByName(prevSgNodeName)
		if err != nil {
			return err
		}
		if !prevSgNode.Fulfilled() {
			continue
		}
		if len(prevSgNode.Children) == 0 {
			woc.addChildNode(ctx, prevSgNodeName, sgNodeName)
			continue
		}
		for _, childID := range prevSgNode.Children {
			for _, outNodeID := range woc.getOutboundNodes(ctx, childID) {
				outNodeName, nameErr := woc.wf.Status.Nodes.GetName(outNodeID)
				if nameErr != nil {
					return nameErr
				}
				woc.addChildNode(ctx, outNodeName, sgNodeName)
			}
		}
	}
	return nil
}
