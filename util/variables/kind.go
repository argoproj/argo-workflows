package variables

type Kind int

const (
	KindGlobal Kind = iota
	KindRuntime
	KindInput
	KindNodeRef
	KindItem
	KindRetry
	KindNodeCtx
	KindMetric
	KindCronWorkflow
)

func (k Kind) String() string {
	return [...]string{"global", "runtime", "input", "node-ref", "item", "retry", "node-ctx", "metric", "cron-workflow"}[k]
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
	TmplAll         TemplateKind = "any"
	TmplContainer   TemplateKind = "container"
	TmplScript      TemplateKind = "script"
	TmplResource    TemplateKind = "resource"
	TmplSteps       TemplateKind = "steps"
	TmplDAG         TemplateKind = "dag"
	TmplData        TemplateKind = "data"
	TmplSuspend     TemplateKind = "suspend"
	TmplHTTP        TemplateKind = "http"
	TmplPlugin       TemplateKind = "plugin"
	TmplExitHandler  TemplateKind = "exit-handler"
	TmplCronWorkflow TemplateKind = "cron-workflow"
)

var AllTemplateKinds = []TemplateKind{
	TmplContainer, TmplScript, TmplResource,
	TmplSteps, TmplDAG, TmplData,
	TmplSuspend, TmplHTTP, TmplPlugin, TmplExitHandler,
	TmplCronWorkflow,
}

var PodKinds = []TemplateKind{TmplContainer, TmplScript, TmplResource}
