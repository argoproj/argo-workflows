// Package keys is the single location where every workflow variable is
// declared. Each package-level var is a handle produced by variables.Define
// and is the only object through which code may read or write that variable.
package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// allPhases is the set at which always-available globals are in scope.
var allPhases = []v.LifecyclePhase{v.PhWorkflowStart, v.PhPreDispatch, v.PhDuringExecute, v.PhExitHandler}

// global declares a workflow-level variable visible at every phase, in any template kind.
func global(template, valueType, description string) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindGlobal, ValueType: valueType,
		AppliesTo: []v.TemplateKind{v.TmplAll}, Phases: allPhases, Description: description,
	})
}

// runtime declares a workflow-level variable whose value is only meaningful
// after the workflow has started (status / duration / failures).
func runtime(template, valueType, description string, phases ...v.LifecyclePhase) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindRuntime, ValueType: valueType,
		AppliesTo: []v.TemplateKind{v.TmplAll}, Phases: phases, Description: description,
	})
}

// workflow.* identity (always available)
var (
	WorkflowName               = global("workflow.name", "string", "Workflow object name")
	WorkflowNamespace          = global("workflow.namespace", "string", "Workflow namespace")
	WorkflowUID                = global("workflow.uid", "string", "Workflow UID")
	WorkflowCreationTimestamp  = global("workflow.creationTimestamp", "string", "RFC3339 creation timestamp")
	WorkflowMainEntrypoint     = global("workflow.mainEntrypoint", "string", "spec.entrypoint")
	WorkflowServiceAccountName = global("workflow.serviceAccountName", "string", "Effective service account name")
	WorkflowPriority           = global("workflow.priority", "string", "Workflow priority. Conditional — resolves only when spec.priority is set; otherwise both lint (validate.go:250-252) and runtime (operator.go:662-664) treat the reference as undefined (no empty/zero fallback).")
)

// workflow.parameters.*
var (
	WorkflowParametersByName = global("workflow.parameters.<name>", "string",
		"Value from spec.arguments.parameters, ConfigMap-resolved if ValueFrom is set")
	WorkflowParametersAll = global("workflow.parameters", "json", "All workflow parameters as a JSON array")
)

// workflow.labels.*, workflow.annotations.*
var (
	WorkflowLabelsByName      = global("workflow.labels.<name>", "string", "Workflow metadata label value")
	WorkflowAnnotationsByName = global("workflow.annotations.<name>", "string", "Workflow metadata annotation value")
)

// workflow.status / duration / failures (runtime)
var (
	WorkflowStatus = runtime("workflow.status", "string",
		"Current workflow phase; final value only at exit handler",
		v.PhPreDispatch, v.PhDuringExecute, v.PhExitHandler)
	WorkflowDuration = runtime("workflow.duration", "string",
		"Elapsed seconds as float string; final at exit handler",
		v.PhPreDispatch, v.PhDuringExecute, v.PhExitHandler)
	WorkflowFailures = runtime("workflow.failures", "json",
		"Failed-node descriptors. Wire format: a strconv.Quote-wrapped JSON string (operator.go:453) — consumers must JSON-decode twice. When no nodes have failed, the value is the literal 6-character string \"null\" (with quotes), not an empty array.",
		v.PhExitHandler)
)
