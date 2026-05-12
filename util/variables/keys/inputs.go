package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// input declares an inputs.* variable available during template execution.
func input(template, valueType, description string, applies []v.TemplateKind) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindInput, ValueType: valueType,
		AppliesTo: applies, Phases: []v.LifecyclePhase{v.PhDuringExecute}, Description: description,
	})
}

// output declares an outputs.* declaration variable (the pod-side declared
// path of an output) available during template execution.
func output(template, valueType, description string, applies []v.TemplateKind) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindOutput, ValueType: valueType,
		AppliesTo: applies, Phases: []v.LifecyclePhase{v.PhDuringExecute}, Description: description,
	})
}

var anyTmpl = []v.TemplateKind{v.TmplAll}

// podKindsOnExit covers every body type that produces a real Pod plus the
// exit-handler context (an onExit template that is itself one of those
// pod-producing types inherits all pod-side variables: pod.name, mount
// paths, …). Distinct from PodKinds because retry- and loop-context
// variables must not leak into the exit handler.
//
// The pod-producing set mirrors Template.IsPodType() in pkg/apis/workflow/
// v1alpha1: Container, ContainerSet, Script, Resource, and Data all run
// real Pods, validate.go gates pod-side variables on IsPodType()/IsLeaf(),
// and the runtime substitutes pod.name and the path variables for all of
// them. Empirically verified end-to-end on the cluster for Data and
// ContainerSet.
var podKindsOnExit = []v.TemplateKind{
	v.TmplContainer, v.TmplContainerSet, v.TmplScript, v.TmplResource, v.TmplData,
	v.TmplExitHandler,
}

// inputs.parameters.*, inputs.artifacts.*
var (
	InputsParameterByName    = input("inputs.parameters.<name>", "string", "Resolved input parameter value", anyTmpl)
	InputsParametersAll      = input("inputs.parameters", "json", "All input parameters as a JSON array", anyTmpl)
	InputsArtifactByName     = input("inputs.artifacts.<name>", "wfv1.Artifact", "Input artifact object (for fromExpression use)", anyTmpl)
	InputsArtifactPathByName = input("inputs.artifacts.<name>.path", "string", "Mount path of the input artifact inside the pod", podKindsOnExit)
)

// outputs.*.path — declared output-side paths (pod-side).
var (
	OutputsArtifactPathByName  = output("outputs.artifacts.<name>.path", "string", "Declared output artifact path for the current template (pod side)", podKindsOnExit)
	OutputsParameterPathByName = output("outputs.parameters.<name>.path", "string", "Declared output parameter path for the current template (pod side)", podKindsOnExit)
)
