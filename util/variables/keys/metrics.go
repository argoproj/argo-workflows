package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// metric declares a variable that is only in scope while a Prometheus metric
// expression is being evaluated. These names overlap with no other lifecycle
// phase: there is no "during-execute" `duration` or `status` available to a
// general template body.
func metric(template, description string) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindMetric, ValueType: "string",
		AppliesTo: anyTmpl, Phases: []v.LifecyclePhase{v.PhMetricEmission},
		Description: description,
	})
}

// metric-emission scope: bare current-node references valid only inside a
// Prometheus metric expression. They do not leak into the regular template
// substitution scope.
var (
	MetricDuration                = metric("duration", "Current node's elapsed duration in seconds")
	MetricStatus                  = metric("status", "Current node's phase")
	MetricExitCode                = metric("exitCode", "Current node's container exit code")
	MetricResourcesDurationByName = metric("resourcesDuration.<resource>", "Current node's resource duration in seconds, keyed by Kubernetes resource name (e.g. cpu, memory)")
	MetricOutputsResult           = metric("outputs.result", "Current node's captured stdout (metric scope only)")
	MetricOutputsParameterByName  = metric("outputs.parameters.<name>", "Current node's named output parameter value (metric scope only)")
)
