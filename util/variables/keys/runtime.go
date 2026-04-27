package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// ─────────────────────────── workflow.creationTimestamp formatters (strftime)

var (
	// WorkflowCreationTimestampFmt covers every %<char> formatter registered
	// in util/strftime. Writers enumerate FormatChars and call Set with the
	// single-letter placeholder.
	WorkflowCreationTimestampFmt = v.Define(v.Spec{
		Template:    "workflow.creationTimestamp.<fmt>",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "strftime-formatted workflow creation time; <fmt> is one of the chars in util/strftime",
	})
	WorkflowCreationTimestampUnix = v.Define(v.Spec{
		Template:    "workflow.creationTimestamp.s",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "Workflow creation time as Unix seconds",
	})
	WorkflowCreationTimestampRFC3339 = v.Define(v.Spec{
		Template:    "workflow.creationTimestamp.RFC3339",
		Kind:        v.KindGlobal,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "Workflow creation time as RFC3339",
	})
)

// ─────────────────────────── cron-scheduled workflows

var WorkflowScheduledTime = v.Define(v.Spec{
	Template:    "workflow.scheduledTime",
	Kind:        v.KindGlobal,
	ValueType:   "string",
	AppliesTo:   []v.TemplateKind{v.TmplAll},
	Phases:      allPhases(),
	Description: "Scheduled time for cron-triggered workflows (from annotation)",
})

// ─────────────────────────── alternative whole-parameters JSON form

// The legacy code writes the parameters JSON twice under two keys. We mirror
// that here so the dual-write is faithful; eventually one of these is
// deprecated.
var WorkflowParametersJSON = v.Define(v.Spec{
	Template:    "workflow.parameters.json",
	Kind:        v.KindGlobal,
	ValueType:   "json",
	AppliesTo:   []v.TemplateKind{v.TmplAll},
	Phases:      allPhases(),
	Description: "All workflow parameters as a JSON array (alias for workflow.parameters)",
})

// ─────────────────────────── whole-JSON labels / annotations forms

var (
	WorkflowAnnotationsAll = v.Define(v.Spec{
		Template:    "workflow.annotations",
		Kind:        v.KindGlobal,
		ValueType:   "json",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "All workflow annotations as a JSON object (deprecated — use workflow.annotations.json)",
	})
	WorkflowAnnotationsJSON = v.Define(v.Spec{
		Template:    "workflow.annotations.json",
		Kind:        v.KindGlobal,
		ValueType:   "json",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "All workflow annotations as a JSON object",
	})
	WorkflowLabelsAll = v.Define(v.Spec{
		Template:    "workflow.labels",
		Kind:        v.KindGlobal,
		ValueType:   "json",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "All workflow labels as a JSON object (deprecated — use workflow.labels.json)",
	})
	WorkflowLabelsJSON = v.Define(v.Spec{
		Template:    "workflow.labels.json",
		Kind:        v.KindGlobal,
		ValueType:   "json",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      allPhases(),
		Description: "All workflow labels as a JSON object",
	})
)
