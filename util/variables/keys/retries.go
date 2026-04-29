package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// retries.* — inside retry-strategy templates.
// Bound only when retryStrategy is set; PhDuringExecute is implied and
// not listed separately, for the same reasons as item / item.<key>.
func retry(template, description string) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindRetry, ValueType: "string", AppliesTo: v.PodKinds,
		Phases:      []v.LifecyclePhase{v.PhInsideRetry},
		Description: description,
	})
}

var (
	Retries             = retry("retries", "0-based retry attempt index")
	RetriesLastExitCode = retry("lastRetry.exitCode", "Exit code of the previous attempt (or 0 on first attempt)")
	RetriesLastStatus   = retry("lastRetry.status", "Phase of the previous attempt (or empty on first)")
	RetriesLastDuration = retry("lastRetry.duration", "Duration of the previous attempt in seconds")
	RetriesLastMessage  = retry("lastRetry.message", "Message of the previous attempt")
)
