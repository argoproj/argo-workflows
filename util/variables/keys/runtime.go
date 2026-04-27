package keys

// workflow.creationTimestamp formatters (strftime).
var (
	WorkflowCreationTimestampFmt = global("workflow.creationTimestamp.<fmt>", "string",
		"strftime-formatted workflow creation time; `<fmt>` is one of the chars in util/strftime")
	WorkflowCreationTimestampUnix    = global("workflow.creationTimestamp.s", "string", "Workflow creation time as Unix seconds")
	WorkflowCreationTimestampRFC3339 = global("workflow.creationTimestamp.RFC3339", "string", "Workflow creation time as RFC3339")
)

// WorkflowScheduledTime is the scheduled time for cron-triggered workflows.
var WorkflowScheduledTime = global("workflow.scheduledTime", "string",
	"Scheduled time for cron-triggered workflows (from annotation)")

// WorkflowParametersJSON is the alternative whole-parameters JSON form.
// Legacy code writes both this and WorkflowParametersAll.
var WorkflowParametersJSON = global("workflow.parameters.json", "json",
	"All workflow parameters as a JSON array (alias for workflow.parameters)")

// Whole-JSON labels / annotations forms.
var (
	WorkflowAnnotationsAll  = global("workflow.annotations", "json", "All workflow annotations as a JSON object (deprecated — use workflow.annotations.json)")
	WorkflowAnnotationsJSON = global("workflow.annotations.json", "json", "All workflow annotations as a JSON object")
	WorkflowLabelsAll       = global("workflow.labels", "json", "All workflow labels as a JSON object (deprecated — use workflow.labels.json)")
	WorkflowLabelsJSON      = global("workflow.labels.json", "json", "All workflow labels as a JSON object")
)
