package variables

// Kind is a coarse categorisation of a variable, used for catalog filtering
// and documentation.
type Kind int

const (
	KindGlobal  Kind = iota // workflow.*
	KindRuntime             // workflow.status, workflow.duration, workflow.failures
	KindInput               // inputs.parameters.*, inputs.artifacts.*
	KindNodeRef             // steps.<name>.*, tasks.<name>.*
	KindItem                // item, item.<key>
	KindRetry               // retries, retries.last*
	KindNodeCtx             // pod.name, node.name, steps.name, tasks.name
)

func (k Kind) String() string {
	switch k {
	case KindGlobal:
		return "global"
	case KindRuntime:
		return "runtime"
	case KindInput:
		return "input"
	case KindNodeRef:
		return "node-ref"
	case KindItem:
		return "item"
	case KindRetry:
		return "retry"
	case KindNodeCtx:
		return "node-ctx"
	}
	return "unknown"
}

// LifecyclePhase is a coarse point-in-time within one workflow execution,
// used to document when a variable is in scope.
type LifecyclePhase string

const (
	// PhWorkflowStart — globals populated once up front (before any template runs).
	PhWorkflowStart LifecyclePhase = "workflow-start"
	// PhPreDispatch — immediately before a template's pod is created; pod.name,
	// node.name, steps.name / tasks.name are set.
	PhPreDispatch LifecyclePhase = "pre-dispatch"
	// PhDuringExecute — inside a template body. inputs.* are bound.
	PhDuringExecute LifecyclePhase = "during-execute"
	// PhInsideLoop — inside a withItems/withParam expansion (item, item.<key>).
	PhInsideLoop LifecyclePhase = "inside-loop"
	// PhInsideRetry — inside a retryStrategy template (retries.*).
	PhInsideRetry LifecyclePhase = "inside-retry"
	// PhAfterNodeInit — once a node has been initialised and has an ID / phase
	// assigned. This is the earliest a reference like steps.X.id works.
	PhAfterNodeInit LifecyclePhase = "after-node-init"
	// PhAfterPodStart — once the node's pod has started running: startedAt, ip,
	// hostNodeName are populated. Earlier than complete.
	PhAfterPodStart LifecyclePhase = "after-pod-start"
	// PhAfterNodeComplete — once a node has finished (any terminal phase):
	// finishedAt and exitCode are populated.
	PhAfterNodeComplete LifecyclePhase = "after-node-complete"
	// PhAfterNodeSucceeded — once a node has finished with Succeeded phase:
	// outputs.result / outputs.parameters.* / outputs.artifacts.* are populated.
	PhAfterNodeSucceeded LifecyclePhase = "after-node-succeeded"
	// PhAfterLoop — once every child of a withItems/withParam group has
	// completed; aggregated outputs appear.
	PhAfterLoop LifecyclePhase = "after-loop"
	// PhExitHandler — the onExit template. workflow.status / workflow.failures /
	// workflow.duration are at their final values. Note: any variable with an
	// earlier phase is also visible here (scope accumulates), so phases that
	// are a strict superset of exit-handler omit it.
	PhExitHandler LifecyclePhase = "exit-handler"
)

// TemplateKind categorises a wfv1.Template by its body type so docs can
// answer "inside a Script template, what variables do I see?"
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
	TmplPlugin      TemplateKind = "plugin"
	TmplExitHandler TemplateKind = "exit-handler"
)

// AllTemplateKinds is the canonical ordering used by the doc generator.
var AllTemplateKinds = []TemplateKind{
	TmplContainer, TmplScript, TmplResource,
	TmplSteps, TmplDAG, TmplData,
	TmplSuspend, TmplHTTP, TmplPlugin,
	TmplExitHandler,
}

// PodKinds is the subset whose nodes produce a pod (and therefore have
// pod.name available).
var PodKinds = []TemplateKind{TmplContainer, TmplScript, TmplResource}
