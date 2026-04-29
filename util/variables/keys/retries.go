package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// retryableKinds are the template kinds that practice has shown actually
// honour retryStrategy and substitute {{retries}} / {{lastRetry.*}} into
// their bodies on each attempt. The validator
// (workflow/validate/validate.go:490) injects retries / lastRetry.* into
// the per-template scope for any tmpl.RetryStrategy != nil regardless of
// kind, and operator.go:2393-2419 substitutes them into processedTmpl.
// Empirically verified end-to-end (initial + 2 retries, per-attempt
// substitution observed) for Container, ContainerSet, Script, Resource,
// Steps, DAG, HTTP, Data, and Plugin templates.
//
// Suspend is intentionally excluded: lint accepts retryStrategy on a
// suspend template, but suspend has no realistic failure path (the
// duration just elapses; explicit termination bypasses retries), so
// retry variables never actually bind in practice.
var retryableKinds = []v.TemplateKind{
	v.TmplContainer, v.TmplContainerSet, v.TmplScript, v.TmplResource,
	v.TmplSteps, v.TmplDAG,
	v.TmplHTTP, v.TmplData, v.TmplPlugin,
}

// retries.* — inside retry-strategy templates.
// Bound only when retryStrategy is set; PhDuringExecute is implied and
// not listed separately, for the same reasons as item / item.<key>.
func retry(template, description string) *v.Key {
	return v.Define(v.Spec{
		Template: template, Kind: v.KindRetry, ValueType: "string", AppliesTo: retryableKinds,
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
