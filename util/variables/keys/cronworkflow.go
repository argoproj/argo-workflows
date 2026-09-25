package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// cronVar declares a CronWorkflow expression variable. These are bound when
// the cron operator evaluates spec.when or spec.stopStrategy.expression on a
// CronWorkflow object — they are not in scope inside any template body.
func cronVar(template, valueType, description string) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindCronWorkflow, ValueType: valueType,
		AppliesTo:   []v.TemplateKind{v.TmplCronWorkflow},
		Phases:      []v.LifecyclePhase{v.PhCronEval},
		Description: description,
	})
}

// cronworkflow.* — CronWorkflow expression scope.
var (
	CronWorkflowName              = cronVar("cronworkflow.name", "string", "CronWorkflow object name")
	CronWorkflowNamespace         = cronVar("cronworkflow.namespace", "string", "CronWorkflow namespace")
	CronWorkflowLabels            = cronVar("cronworkflow.labels", "map", "CronWorkflow labels as a map; supports nested key access (cronworkflow.labels.foo)")
	CronWorkflowAnnotations       = cronVar("cronworkflow.annotations", "map", "CronWorkflow annotations as a map; supports nested key access (cronworkflow.annotations.foo)")
	CronWorkflowLabelsJSON        = cronVar("cronworkflow.labels.json", "json", "CronWorkflow labels as a JSON object")
	CronWorkflowAnnotationsJSON   = cronVar("cronworkflow.annotations.json", "json", "CronWorkflow annotations as a JSON object")
	CronWorkflowFailed            = cronVar("cronworkflow.failed", "int", "Count of failed child Workflows")
	CronWorkflowSucceeded         = cronVar("cronworkflow.succeeded", "int", "Count of succeeded child Workflows")
	CronWorkflowLastScheduledTime = cronVar("cronworkflow.lastScheduledTime", "*time.Time", "Time the cron last triggered, or nil before the first run")
)
