package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// input declares an inputs.* variable available during template execution.
func input(template, valueType, description string, applies []v.TemplateKind) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindInput, ValueType: valueType,
		AppliesTo: applies, Phases: []v.LifecyclePhase{v.PhDuringExecute}, Description: description,
	})
}

var anyTmpl = []v.TemplateKind{v.TmplAll}

// inputs.parameters.*, inputs.artifacts.*
var (
	InputsParameterByName    = input("inputs.parameters.<name>", "string", "Resolved input parameter value", anyTmpl)
	InputsParametersAll      = input("inputs.parameters", "json", "All input parameters as a JSON array", anyTmpl)
	InputsArtifactByName     = input("inputs.artifacts.<name>", "wfv1.Artifact", "Input artifact object (for fromExpression use)", anyTmpl)
	InputsArtifactPathByName = input("inputs.artifacts.<name>.path", "string", "Mount path of the input artifact inside the pod", v.PodKinds)
)

// outputs.*.path — declared input-side paths.
var (
	OutputsArtifactPathByName  = input("outputs.artifacts.<name>.path", "string", "Declared output artifact path for the current template (pod side)", v.PodKinds)
	OutputsParameterPathByName = input("outputs.parameters.<name>.path", "string", "Declared output parameter path for the current template (pod side)", v.PodKinds)
)
