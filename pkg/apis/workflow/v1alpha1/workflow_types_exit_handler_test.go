package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// exitNode builds a node the way initializeNode does: an empty BoundaryID
// keeps DisplayName equal to Name (workflow-level onExit and its retry
// attempts); a boundary trims the boundary name and the steps/DAG separator.
func exitNode(id, name, display, boundary string, typ NodeType, children ...string) NodeStatus {
	return NodeStatus{
		ID:          id,
		Name:        name,
		DisplayName: display,
		BoundaryID:  boundary,
		Type:        typ,
		Phase:       NodePending,
		Children:    children,
	}
}

func TestIsPartOfExitHandler(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Workflow-level onExit wrapped in a Retry node. Attempts keep the retry
	// node's empty BoundaryID and are named "<wf>.onExit(N)", so DisplayName
	// does not end in ".onExit". The Retry node lists them as children.
	// A lifecycle hooks.exit is reconciled as this same "<wf>.onExit" node
	// (GenerateOnExitNodeName), flagged Hooked — not as "<wf>.hooks.exit".
	exitRetry := exitNode("exit", "wf.onExit", "wf.onExit", "", NodeTypeRetry, "a0", "a1", "hook")
	exitRetry.NodeFlag = &NodeFlag{Hooked: true}
	attempt0 := exitNode("a0", "wf.onExit(0)", "wf.onExit(0)", "", NodeTypePod)
	attempt0.NodeFlag = &NodeFlag{Hooked: true, Retried: true}
	attempt1 := exitNode("a1", "wf.onExit(1)", "wf.onExit(1)", "", NodeTypePod)
	attempt1.NodeFlag = &NodeFlag{Hooked: true, Retried: true}
	// A hook of the exit template is a child of the Retry node and shares its
	// empty boundary. It is part of the handler even though its name is
	// ".hooks.running", not ".onExit".
	exitHook := exitNode("hook", "wf.onExit.hooks.running", "wf.onExit.hooks.running", "", NodeTypePod)
	exitHook.NodeFlag = &NodeFlag{Hooked: true}

	// Non-exit retry of the entrypoint. "<wf>(0)" must not match ".onExit(0)".
	rootRetry := exitNode("root", "wf", "wf", "", NodeTypeRetry, "root0")
	rootAttempt := exitNode("root0", "wf(0)", "wf(0)", "", NodeTypePod)
	rootAttempt.NodeFlag = &NodeFlag{Retried: true}
	runningHook := exitNode("run", "wf.hooks.running", "wf.hooks.running", "", NodeTypePod)
	runningHook.NodeFlag = &NodeFlag{Hooked: true}

	// Steps boundary. A step onExit's BoundaryID is the steps node, not the
	// retry node, so walking BoundaryID never sees the ".onExit" node.
	// Display names are what initializeNode produces: trim the boundary name
	// "wf", then strip the leading "[0].".
	steps := exitNode("steps", "wf", "wf", "", NodeTypeSteps, "group")
	group := exitNode("group", "wf[0]", "[0]", "steps", NodeTypeStepGroup, "step")
	step := exitNode("step", "wf[0].A", "A", "steps", NodeTypePod, "stepExit")
	stepExit := exitNode("stepExit", "wf[0].A.onExit", "A.onExit", "steps", NodeTypeRetry, "stepExit0")
	stepExit.NodeFlag = &NodeFlag{Hooked: true}
	stepExit0 := exitNode("stepExit0", "wf[0].A.onExit(0)", "A.onExit(0)", "steps", NodeTypePod)
	stepExit0.NodeFlag = &NodeFlag{Hooked: true, Retried: true}
	stepRetry := exitNode("stepRetry", "wf[0].B", "B", "steps", NodeTypeRetry, "stepRetry0")
	stepRetry0 := exitNode("stepRetry0", "wf[0].B(0)", "B(0)", "steps", NodeTypePod)
	stepRetry0.NodeFlag = &NodeFlag{Retried: true}

	// DAG task onExit, same boundary relationship. When task has a retryStrategy,
	// getOutboundNodes returns its last child (wf.task.onExit), so a dependent
	// DAG task (wf.nextTask) is linked in taskExit.Children as an outbound edge.
	// That downstream task belongs to the DAG boundary, not to the exit handler.
	dag := exitNode("dag", "wf", "wf", "", NodeTypeDAG, "task", "nextTask")
	task := exitNode("task", "wf.task", "task", "dag", NodeTypeRetry, "task0", "taskExit")
	task0 := exitNode("task0", "wf.task(0)", "task(0)", "dag", NodeTypePod)
	task0.NodeFlag = &NodeFlag{Retried: true}
	taskExit := exitNode("taskExit", "wf.task.onExit", "task.onExit", "dag", NodeTypeRetry, "taskExit0", "nextTask")
	taskExit.NodeFlag = &NodeFlag{Hooked: true}
	taskExit0 := exitNode("taskExit0", "wf.task.onExit(0)", "task.onExit(0)", "dag", NodeTypePod)
	taskExit0.NodeFlag = &NodeFlag{Hooked: true, Retried: true}
	nextTask := exitNode("nextTask", "wf.nextTask", "nextTask", "dag", NodeTypeRetry, "nextTask0")
	nextTask0 := exitNode("nextTask0", "wf.nextTask(0)", "nextTask(0)", "dag", NodeTypePod)
	nextTask0.NodeFlag = &NodeFlag{Retried: true}

	// Similarly, a next StepGroup hung off a step's exit Retry node is not part
	// of the exit handler.
	stepExit.Children = append(stepExit.Children, "nextGroup")
	nextGroup := exitNode("nextGroup", "wf[1]", "[1]", "steps", NodeTypeStepGroup)

	// Retried workflow onExit whose template is a DAG. The attempt is the DAG
	// node and a child of the Retry node; a task inside it is its own Retry,
	// and that attempt's pod has the DAG as its boundary. A container of the
	// pod hangs off the pod.
	nestedExit := exitNode("nestedExit", "nested.onExit", "nested.onExit", "", NodeTypeRetry, "onExitDAG")
	onExitDAG := exitNode("onExitDAG", "nested.onExit(0)", "nested.onExit(0)", "", NodeTypeDAG, "notifyRetry")
	notifyRetry := exitNode("notifyRetry", "nested.onExit(0).notify", "notify", "onExitDAG", NodeTypeRetry, "notify0")
	notify0 := exitNode("notify0", "nested.onExit(0).notify(0)", "notify(0)", "onExitDAG", NodeTypePod, "notifyCtr")
	notify0.NodeFlag = &NodeFlag{Retried: true}
	notifyCtr := exitNode("notifyCtr", "nested.onExit(0).notify(0).main", "main", "notify0", NodeTypeContainer)

	// onExit steps with no retry of its own: children are found by BoundaryID.
	plainExit := exitNode("plainExit", "wf.onExit", "wf.onExit", "", NodeTypeSteps, "plainGroup")
	plainGroup := exitNode("plainGroup", "wf.onExit[0]", "[0]", "plainExit", NodeTypeStepGroup, "plainStep")
	plainStep := exitNode("plainStep", "wf.onExit[0].step", "step", "plainExit", NodeTypePod)

	nodes := Nodes{}
	for _, n := range []NodeStatus{
		exitRetry, attempt0, attempt1, exitHook,
		rootRetry, rootAttempt, runningHook,
		steps, group, step, stepExit, stepExit0, nextGroup, stepRetry, stepRetry0,
		dag, task, task0, taskExit, taskExit0, nextTask, nextTask0,
		nestedExit, onExitDAG, notifyRetry, notify0, notifyCtr,
		plainExit, plainGroup, plainStep,
	} {
		nodes[n.ID] = n
	}

	cases := []struct {
		id   string
		want bool
	}{
		{"exit", true},
		{"a0", true},
		{"a1", true},
		{"hook", true},
		{"root", false},
		{"root0", false},
		{"run", false},
		{"stepExit", true},
		{"stepExit0", true},
		{"nextGroup", false},
		{"step", false},
		{"stepRetry", false},
		{"stepRetry0", false},
		{"taskExit", true},
		{"taskExit0", true},
		{"task", false},
		{"task0", false},
		{"nextTask", false},
		{"nextTask0", false},
		{"nestedExit", true},
		{"onExitDAG", true},
		{"notifyRetry", true},
		{"notify0", true},
		{"notifyCtr", true},
		{"plainExit", true},
		{"plainStep", true},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			node, err := nodes.Get(tc.id)
			require.NoError(t, err)
			assert.Equal(t, tc.want, node.IsPartOfExitHandler(ctx, nodes))
		})
	}
}
