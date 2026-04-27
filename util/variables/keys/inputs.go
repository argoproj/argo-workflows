package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// ─────────────────────────── inputs.parameters.*, inputs.artifacts.*
//
// Bound during template execution. Every template kind has inputs.

var (
	InputsParameterByName = v.Define(v.Spec{
		Template:    "inputs.parameters.<name>",
		Kind:        v.KindInput,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhDuringExecute},
		Description: "Resolved input parameter value",
	})
	InputsParametersAll = v.Define(v.Spec{
		Template:    "inputs.parameters",
		Kind:        v.KindInput,
		ValueType:   "json",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhDuringExecute},
		Description: "All input parameters as a JSON array",
	})
	InputsArtifactByName = v.Define(v.Spec{
		Template:    "inputs.artifacts.<name>",
		Kind:        v.KindInput,
		ValueType:   "wfv1.Artifact",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhDuringExecute},
		Description: "Input artifact object (for fromExpression use)",
	})
	InputsArtifactPathByName = v.Define(v.Spec{
		Template:    "inputs.artifacts.<name>.path",
		Kind:        v.KindInput,
		ValueType:   "string",
		AppliesTo:   v.PodKinds,
		Phases:      []v.LifecyclePhase{v.PhDuringExecute},
		Description: "Mount path of the input artifact inside the pod",
	})
)

// ─────────────────────────── outputs.*.path — declared input-side paths

var (
	OutputsArtifactPathByName = v.Define(v.Spec{
		Template:    "outputs.artifacts.<name>.path",
		Kind:        v.KindInput,
		ValueType:   "string",
		AppliesTo:   v.PodKinds,
		Phases:      []v.LifecyclePhase{v.PhDuringExecute},
		Description: "Declared output artifact path for the current template (pod side)",
	})
	OutputsParameterPathByName = v.Define(v.Spec{
		Template:    "outputs.parameters.<name>.path",
		Kind:        v.KindInput,
		ValueType:   "string",
		AppliesTo:   v.PodKinds,
		Phases:      []v.LifecyclePhase{v.PhDuringExecute},
		Description: "Declared output parameter path for the current template (pod side)",
	})
)
