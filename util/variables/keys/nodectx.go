package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// pod.name / node.name / steps.name / tasks.name — context names available
// just before a pod is dispatched and during execution.
func nodeCtx(template, description string, applies []v.TemplateKind) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindNodeCtx, ValueType: "string", AppliesTo: applies,
		Phases:      []v.LifecyclePhase{v.PhPreDispatch, v.PhDuringExecute},
		Description: description,
	})
}

var (
	// pod.name is in scope for any pod-producing body, including an
	// exit-handler template whose body is itself container/script/resource.
	PodName   = nodeCtx("pod.name", "Computed pod name for pod-producing templates", podKindsOnExit)
	NodeName  = nodeCtx("node.name", "Full node name", anyTmpl)
	StepsName = nodeCtx("steps.name", "Name of the current step (inside a Steps template body)", []v.TemplateKind{v.TmplSteps})
	TasksName = nodeCtx("tasks.name", "Name of the current task (inside a DAG template body)", []v.TemplateKind{v.TmplDAG})
)
