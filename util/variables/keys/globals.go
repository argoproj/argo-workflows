// Package keys is the single location where every workflow variable is
// declared. Each package-level var is a handle produced by variables.Define
// and is the only object through which code may read or write that variable.
//
// Adding a new workflow variable is one commit in this file plus one import
// at the write site. Removing a variable is one deletion here — references
// to the handle become compile errors, making dead catalog entries
// impossible.
package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// ─────────────────────────── workflow.* identity (always available)

var (
	WorkflowName = v.Define(v.Spec{
		Template:    "workflow.name",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "Workflow object name",
	})
	WorkflowNamespace = v.Define(v.Spec{
		Template:    "workflow.namespace",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "Workflow namespace",
	})
	WorkflowUID = v.Define(v.Spec{
		Template:    "workflow.uid",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "Workflow UID",
	})
	WorkflowCreationTimestamp = v.Define(v.Spec{
		Template:    "workflow.creationTimestamp",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "RFC3339 creation timestamp",
	})
	WorkflowMainEntrypoint = v.Define(v.Spec{
		Template:    "workflow.mainEntrypoint",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "spec.entrypoint",
	})
	WorkflowServiceAccountName = v.Define(v.Spec{
		Template:    "workflow.serviceAccountName",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "Effective service account name",
	})
	WorkflowPriority = v.Define(v.Spec{
		Template:    "workflow.priority",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "Workflow priority",
	})
)

// ─────────────────────────── workflow.parameters.*

var (
	WorkflowParametersByName = v.Define(v.Spec{
		Template:    "workflow.parameters.<name>",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "Value from spec.arguments.parameters, ConfigMap-resolved if ValueFrom is set",
	})
	WorkflowParametersAll = v.Define(v.Spec{
		Template:    "workflow.parameters",
		Kind:        v.KindGlobal,
		ValueType:   "json",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "All workflow parameters as a JSON array",
	})
)

// ─────────────────────────── workflow.labels.*, workflow.annotations.*

var (
	WorkflowLabelsByName = v.Define(v.Spec{
		Template:    "workflow.labels.<name>",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "Workflow metadata label value",
	})
	WorkflowAnnotationsByName = v.Define(v.Spec{
		Template:    "workflow.annotations.<name>",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "Workflow metadata annotation value",
	})
)

// ─────────────────────────── workflow.status / duration / failures (runtime)

var (
	WorkflowStatus = v.Define(v.Spec{
		Template:    "workflow.status",
		Kind:        v.KindRuntime,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhPreDispatch, v.PhDuringExecute, v.PhExitHandler},
		Description: "Current workflow phase; final value only at exit handler",
	})
	WorkflowDuration = v.Define(v.Spec{
		Template:    "workflow.duration",
		Kind:        v.KindRuntime,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhPreDispatch, v.PhDuringExecute, v.PhExitHandler},
		Description: "Elapsed seconds as float string; final at exit handler",
	})
	WorkflowFailures = v.Define(v.Spec{
		Template:    "workflow.failures",
		Kind:        v.KindRuntime,
		ValueType:   "json",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhExitHandler},
		Description: "JSON array of failed node descriptors; populated when any node failed",
	})
)

// allPhases returns the set of phases at which always-available globals
// are in scope. A helper because we call it many times.
func allPhases() []v.LifecyclePhase {
	return []v.LifecyclePhase{
		v.PhWorkflowStart, v.PhPreDispatch, v.PhDuringExecute, v.PhExitHandler,
	}
}
