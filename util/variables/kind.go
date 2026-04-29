package variables

type Kind int

const (
	KindGlobal Kind = iota
	KindRuntime
	KindInput
	KindOutput
	KindNodeRef
	KindItem
	KindRetry
	KindNodeCtx
	KindMetric
	KindCronWorkflow
)

func (k Kind) String() string {
	return [...]string{"global", "runtime", "input", "output", "node-ref", "item", "retry", "node-ctx", "metric", "cron-workflow"}[k]
}

// LifecyclePhase is a coarse point-in-time within one workflow execution.
// Phase names are surfaced verbatim in the generated docs.
type LifecyclePhase string

const (
	PhWorkflowStart      LifecyclePhase = "workflow-start"
	PhPreDispatch        LifecyclePhase = "pre-dispatch"
	PhDuringExecute      LifecyclePhase = "during-execute"
	PhInsideLoop         LifecyclePhase = "inside-loop"
	PhInsideRetry        LifecyclePhase = "inside-retry"
	PhAfterNodeInit      LifecyclePhase = "after-node-init"
	PhAfterPodStart      LifecyclePhase = "after-pod-start"
	PhAfterNodeComplete  LifecyclePhase = "after-node-complete"
	PhAfterNodeSucceeded LifecyclePhase = "after-node-succeeded"
	PhAfterLoop          LifecyclePhase = "after-loop"
	PhExitHandler        LifecyclePhase = "exit-handler"
	PhMetricEmission     LifecyclePhase = "metric-emission"
	PhCronEval           LifecyclePhase = "cron-eval"
)

// TemplateKind categorises a wfv1.Template by its body type.
type TemplateKind string

const (
	TmplAll          TemplateKind = "any"
	TmplContainer    TemplateKind = "container"
	TmplContainerSet TemplateKind = "container-set"
	TmplScript       TemplateKind = "script"
	TmplResource     TemplateKind = "resource"
	TmplSteps        TemplateKind = "steps"
	TmplDAG          TemplateKind = "dag"
	TmplData         TemplateKind = "data"
	TmplSuspend      TemplateKind = "suspend"
	TmplHTTP         TemplateKind = "http"
	TmplPlugin       TemplateKind = "plugin"
	TmplExitHandler  TemplateKind = "exit-handler"
	TmplCronWorkflow TemplateKind = "cron-workflow"
)

var AllTemplateKinds = []TemplateKind{
	TmplContainer, TmplContainerSet, TmplScript, TmplResource,
	TmplSteps, TmplDAG, TmplData,
	TmplSuspend, TmplHTTP, TmplPlugin, TmplExitHandler,
	TmplCronWorkflow,
}

// PodKinds are the body types that produce a real Kubernetes Pod.
// Mirrors `Template.IsPodType()` in pkg/apis/workflow/v1alpha1.
var PodKinds = []TemplateKind{TmplContainer, TmplContainerSet, TmplScript, TmplResource, TmplData}

// ReachablePhases returns the lifecycle phases that can actually fire from
// inside a template body of the given kind. Used by the doc generator to
// gate matrix `•` marks: a variable is only shown reachable from a
// TemplateKind column if at least one of its declared phases is in this
// set. This prevents "narrow" phases (PhInsideLoop, PhInsideRetry,
// PhExitHandler, PhCronEval) from leaking into columns that never enter
// those phases — e.g. `item` showing up under the exit-handler column.
//
// TmplExitHandler is treated as a context, not a body type: any leaf body
// (container/script/etc.) can serve as an onExit handler, but the matrix
// renders it as a separate column for catalog readers, so its column
// reflects exit-handler semantics rather than generic body semantics.
func ReachablePhases(k TemplateKind) []LifecyclePhase {
	switch k {
	case TmplCronWorkflow:
		return []LifecyclePhase{PhCronEval}
	case TmplExitHandler:
		return []LifecyclePhase{
			PhWorkflowStart, PhPreDispatch, PhDuringExecute,
			PhAfterNodeInit, PhAfterPodStart, PhAfterNodeComplete,
			PhAfterNodeSucceeded, PhAfterLoop,
			PhExitHandler, PhMetricEmission,
		}
	case TmplContainer, TmplContainerSet, TmplScript, TmplResource:
		return []LifecyclePhase{
			PhWorkflowStart, PhPreDispatch, PhDuringExecute,
			PhInsideLoop, PhInsideRetry, PhMetricEmission,
		}
	case TmplSteps, TmplDAG:
		return []LifecyclePhase{
			PhWorkflowStart, PhPreDispatch, PhDuringExecute,
			PhInsideLoop, PhInsideRetry,
			PhAfterNodeInit, PhAfterPodStart, PhAfterNodeComplete,
			PhAfterNodeSucceeded, PhAfterLoop, PhMetricEmission,
		}
	case TmplData, TmplHTTP, TmplPlugin:
		// PhInsideRetry: retryStrategy demonstrably fires on these kinds and
		// {{retries}} / {{lastRetry.*}} substitute inside their bodies.
		// (Suspend is excluded — lint accepts retryStrategy but suspend has no
		// realistic failure path so retry vars never bind.)
		return []LifecyclePhase{
			PhWorkflowStart, PhPreDispatch, PhDuringExecute,
			PhInsideLoop, PhInsideRetry, PhMetricEmission,
		}
	case TmplSuspend:
		return []LifecyclePhase{
			PhWorkflowStart, PhPreDispatch, PhDuringExecute,
			PhInsideLoop, PhMetricEmission,
		}
	}
	return nil
}
