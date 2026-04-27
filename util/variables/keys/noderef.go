package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// NodeRefKeys bundles every variable that refers to a specific sibling node
// (steps or tasks). The same shape applies to both; we instantiate it twice.
type NodeRefKeys struct {
	ID            *v.Key
	StartedAt     *v.Key
	FinishedAt    *v.Key
	IP            *v.Key
	Status        *v.Key
	HostNodeName  *v.Key
	OutputsResult *v.Key
	ExitCode      *v.Key
	// Parameterised on output name:
	OutputsParameterByName *v.Key
	OutputsArtifactByName  *v.Key
}

// StepsNodeRef and TasksNodeRef share shape; the only difference is the
// prefix in the template. Having them as separate handles keeps the
// correctness-by-construction rule: writing "steps.X.id" vs "tasks.X.id" is
// two different Keys and you pick the right one.
var (
	StepsNodeRef = declareNodeRef("steps", v.TmplSteps)
	TasksNodeRef = declareNodeRef("tasks", v.TmplDAG)
)

func declareNodeRef(pfx string, inKind v.TemplateKind) NodeRefKeys {
	applies := []v.TemplateKind{inKind, v.TmplExitHandler}
	// Each field becomes available at a different sub-phase of the referenced
	// node's lifecycle. PhExitHandler is omitted where an earlier phase is a
	// strict superset (the exit handler sees all accumulated scope).
	def := func(suffix, typ, desc string, phases ...v.LifecyclePhase) *v.Key {
		return v.Define(v.Spec{
			Template:    pfx + ".<name>" + suffix,
			Kind:        v.KindNodeRef,
			ValueType:   typ,
			AppliesTo:   applies,
			Phases:      phases,
			Description: desc,
		})
	}
	return NodeRefKeys{
		ID:            def(".id", "string", "Node ID", v.PhAfterNodeInit),
		Status:        def(".status", "string", "Node phase", v.PhAfterNodeInit),
		StartedAt:     def(".startedAt", "string", "RFC3339 start time", v.PhAfterPodStart),
		IP:            def(".ip", "string", "Pod IP", v.PhAfterPodStart),
		HostNodeName:  def(".hostNodeName", "string", "Underlying k8s node name", v.PhAfterPodStart),
		FinishedAt:    def(".finishedAt", "string", "RFC3339 finish time", v.PhAfterNodeComplete),
		ExitCode:      def(".exitCode", "string", "Container exit code", v.PhAfterNodeComplete),
		OutputsResult: def(".outputs.result", "string", "Captured stdout (non-loop nodes)", v.PhAfterNodeSucceeded),
		OutputsParameterByName: v.Define(v.Spec{
			Template:    pfx + ".<name>.outputs.parameters.<p>",
			Kind:        v.KindNodeRef,
			ValueType:   "string",
			AppliesTo:   applies,
			Phases:      []v.LifecyclePhase{v.PhAfterNodeSucceeded},
			Description: "Named output parameter of the referenced node",
		}),
		OutputsArtifactByName: v.Define(v.Spec{
			Template:    pfx + ".<name>.outputs.artifacts.<a>",
			Kind:        v.KindNodeRef,
			ValueType:   "wfv1.Artifact",
			AppliesTo:   applies,
			Phases:      []v.LifecyclePhase{v.PhAfterNodeSucceeded},
			Description: "Named output artifact of the referenced node",
		}),
	}
}

// ─────────────────────────── Aggregated (withItems / withParam) outputs

type AggregateKeys struct {
	Result            *v.Key // outputs.result (JSON array)
	Parameters        *v.Key // outputs.parameters (JSON array of maps)
	ParameterByName   *v.Key // outputs.parameters.<p> (JSON array of values)
}

var (
	StepsAggregate = declareAggregate("steps", v.TmplSteps)
	TasksAggregate = declareAggregate("tasks", v.TmplDAG)
)

func declareAggregate(pfx string, inKind v.TemplateKind) AggregateKeys {
	applies := []v.TemplateKind{inKind, v.TmplExitHandler}
	// PhAfterLoop is a strict subset of PhExitHandler's visibility; no need
	// to list the latter.
	ph := []v.LifecyclePhase{v.PhAfterLoop}
	return AggregateKeys{
		Result: v.Define(v.Spec{
			Template:    pfx + ".<loopName>.outputs.result",
			Kind:        v.KindNodeRef,
			ValueType:   "json",
			AppliesTo:   applies,
			Phases:      ph,
			Description: "JSON array of child results (withItems/withParam)",
		}),
		Parameters: v.Define(v.Spec{
			Template:    pfx + ".<loopName>.outputs.parameters",
			Kind:        v.KindNodeRef,
			ValueType:   "json",
			AppliesTo:   applies,
			Phases:      ph,
			Description: "JSON array of per-child output-parameter maps",
		}),
		ParameterByName: v.Define(v.Spec{
			Template:    pfx + ".<loopName>.outputs.parameters.<p>",
			Kind:        v.KindNodeRef,
			ValueType:   "json",
			AppliesTo:   applies,
			Phases:      ph,
			Description: "JSON array of values for a named parameter across all children",
		}),
	}
}

// ─────────────────────────── workflow.* outputs (lifted via globalName)

var (
	WorkflowOutputsParameterByName = v.Define(v.Spec{
		Template:    "workflow.outputs.parameters.<name>",
		Kind:        v.KindNodeRef,
		ValueType:   "string",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhDuringExecute, v.PhExitHandler},
		Description: "Global output parameter (lifted via outputs.parameters[*].globalName)",
	})
	WorkflowOutputsArtifactByName = v.Define(v.Spec{
		Template:    "workflow.outputs.artifacts.<name>",
		Kind:        v.KindNodeRef,
		ValueType:   "wfv1.Artifact",
		AppliesTo:   []v.TemplateKind{v.TmplAll},
		Phases:      []v.LifecyclePhase{v.PhDuringExecute, v.PhExitHandler},
		Description: "Global output artifact (lifted via outputs.artifacts[*].globalName)",
	})
)
