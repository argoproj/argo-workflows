package controller

import (
	"context"
	"fmt"

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
	return fmt.Sprintf("[%d].%s", s.groupIndex, s.step.Name)
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
	if s.step.ContinueOn != nil {
		if s.step.ContinueOn.Failed && phase == wfv1.NodeFailed {
			return true
		}
		if s.step.ContinueOn.Error && phase == wfv1.NodeError {
			return true
		}
	}
	return false
}

func (s *StepAdapter) GetHooks() wfv1.LifecycleHooks {
	return s.step.Hooks
}

func (s *StepAdapter) GetExitHook(args wfv1.Arguments) *wfv1.LifecycleHook {
	onExit := s.step.OnExit //nolint:staticcheck // OnExit is deprecated but still honored for backward compatibility
	hasExitHook := (s.step.Hooks != nil && s.step.Hooks.HasExitHook()) || onExit != ""
	if !hasExitHook {
		return nil
	}
	if onExit != "" {
		return &wfv1.LifecycleHook{Template: onExit, Arguments: args}
	}
	return s.step.Hooks.GetExitHook().WithArgs(args)
}

func (s *StepAdapter) Expand(ctx context.Context, scope map[string]string, substitutor dag.Substitutor) ([]dag.Task, error) {
	// Construct a temporary DAGTask to reuse the DAG expansion logic
	dt := &dag.DAGTask{DAGTask: &wfv1.DAGTask{
		Name:         s.GetName(),
		Template:     s.step.Template,
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
	return dt.Expand(ctx, scope, substitutor)
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
		sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
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
		sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
		prevSgNodeName := fmt.Sprintf("%s[%d]", nodeName, i-1)
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
