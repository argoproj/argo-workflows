package keys

import v "github.com/argoproj/argo-workflows/v4/util/variables"

// NodeRefKeys bundles every variable that refers to a sibling node (steps or
// tasks). The same shape applies to both; instantiated twice.
type NodeRefKeys struct {
	ID, Status, StartedAt, IP, HostNodeName *v.Key
	FinishedAt, ExitCode                    *v.Key
	OutputsResult                           *v.Key
	OutputsParameterByName                  *v.Key
	OutputsArtifactByName                   *v.Key
}

var (
	StepsNodeRef = nodeRef("steps", v.TmplSteps)
	TasksNodeRef = nodeRef("tasks", v.TmplDAG)
)

func nodeRef(pfx string, inKind v.TemplateKind) NodeRefKeys {
	applies := []v.TemplateKind{inKind, v.TmplExitHandler}
	def := func(suffix, desc string, phase v.LifecyclePhase) *v.Key {
		return v.Define(v.Spec{
			Template: pfx + ".<name>" + suffix, Kind: v.KindNodeRef, ValueType: "string",
			AppliesTo: applies, Phases: []v.LifecyclePhase{phase}, Description: desc,
		})
	}
	return NodeRefKeys{
		ID:            def(".id", "Node ID", v.PhAfterNodeInit),
		Status:        def(".status", "Node phase", v.PhAfterNodeInit),
		StartedAt:     def(".startedAt", "RFC3339 start time", v.PhAfterPodStart),
		IP:            def(".ip", "Pod IP", v.PhAfterPodStart),
		HostNodeName:  def(".hostNodeName", "Underlying k8s node name", v.PhAfterPodStart),
		FinishedAt:    def(".finishedAt", "RFC3339 finish time", v.PhAfterNodeComplete),
		ExitCode:      def(".exitCode", "Container exit code", v.PhAfterNodeComplete),
		OutputsResult: def(".outputs.result", "Captured stdout (non-loop nodes)", v.PhAfterNodeSucceeded),
		OutputsParameterByName: v.Define(v.Spec{
			Template: pfx + ".<name>.outputs.parameters.<p>", Kind: v.KindNodeRef, ValueType: "string",
			AppliesTo: applies, Phases: []v.LifecyclePhase{v.PhAfterNodeSucceeded},
			Description: "Named output parameter of the referenced node",
		}),
		OutputsArtifactByName: v.Define(v.Spec{
			Template: pfx + ".<name>.outputs.artifacts.<a>", Kind: v.KindNodeRef, ValueType: "wfv1.Artifact",
			AppliesTo: applies, Phases: []v.LifecyclePhase{v.PhAfterNodeSucceeded},
			Description: "Named output artifact of the referenced node",
		}),
	}
}

// AggregateKeys are the per-loop outputs of a withItems/withParam group.
type AggregateKeys struct {
	Result, Parameters, ParameterByName *v.Key
}

var (
	StepsAggregate = aggregate("steps", v.TmplSteps)
	TasksAggregate = aggregate("tasks", v.TmplDAG)
)

func aggregate(pfx string, inKind v.TemplateKind) AggregateKeys {
	applies := []v.TemplateKind{inKind, v.TmplExitHandler}
	ph := []v.LifecyclePhase{v.PhAfterLoop}
	def := func(suffix, desc string) *v.Key {
		return v.Define(v.Spec{
			Template: pfx + ".<loopName>" + suffix, Kind: v.KindNodeRef, ValueType: "json",
			AppliesTo: applies, Phases: ph, Description: desc,
		})
	}
	return AggregateKeys{
		Result:          def(".outputs.result", "JSON array of child results (withItems/withParam)"),
		Parameters:      def(".outputs.parameters", "JSON array of per-child output-parameter maps"),
		ParameterByName: def(".outputs.parameters.<p>", "JSON array of values for a named parameter across all children"),
	}
}

// workflow.outputs.* (lifted via globalName).
var (
	WorkflowOutputsParameterByName = v.Define(v.Spec{
		Template: "workflow.outputs.parameters.<name>", Kind: v.KindNodeRef, ValueType: "string",
		AppliesTo: []v.TemplateKind{v.TmplAll}, Phases: []v.LifecyclePhase{v.PhDuringExecute, v.PhExitHandler},
		Description: "Global output parameter (lifted via outputs.parameters[*].globalName)",
	})
	WorkflowOutputsArtifactByName = v.Define(v.Spec{
		Template: "workflow.outputs.artifacts.<name>", Kind: v.KindNodeRef, ValueType: "wfv1.Artifact",
		AppliesTo: []v.TemplateKind{v.TmplAll}, Phases: []v.LifecyclePhase{v.PhDuringExecute, v.PhExitHandler},
		Description: "Global output artifact (lifted via outputs.artifacts[*].globalName)",
	})
)
