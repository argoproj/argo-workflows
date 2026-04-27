package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// ─────────────────────────── pod.name / node.name / steps.name / tasks.name

var (
	PodName = v.Define(v.Spec{
		Template:    "pod.name",
		Kind:        v.KindNodeCtx,
		ValueType:   "string",
		AppliesTo:   v.PodKinds,
		Phases:      []v.LifecyclePhase{v.PhPreDispatch, v.PhDuringExecute},
		Description: "Computed pod name for pod-producing templates",
	})
	NodeName = v.Define(v.Spec{
		Template:    "node.name",
		Kind:        v.KindNodeCtx,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhPreDispatch, v.PhDuringExecute},
		Description: "Full node name",
	})
	StepsName = v.Define(v.Spec{
		Template:    "steps.name",
		Kind:        v.KindNodeCtx,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplSteps},
		Phases:      []v.LifecyclePhase{v.PhPreDispatch, v.PhDuringExecute},
		Description: "Name of the current step (inside a Steps template body)",
	})
	TasksName = v.Define(v.Spec{
		Template:    "tasks.name",
		Kind:        v.KindNodeCtx,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplDAG},
		Phases:      []v.LifecyclePhase{v.PhPreDispatch, v.PhDuringExecute},
		Description: "Name of the current task (inside a DAG template body)",
	})
)
