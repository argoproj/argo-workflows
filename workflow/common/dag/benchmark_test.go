package dag

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func generateRandomDAG(n int) (*wfv1.Workflow, *wfv1.Template) {
	// Use a fixed seed for reproducibility
	r := rand.New(rand.NewSource(42))

	tasks := make([]wfv1.DAGTask, n)
	for i := range n {
		tasks[i] = wfv1.DAGTask{Name: fmt.Sprintf("task-%d", i)}

		// Add random dependencies from previous nodes to ensure DAG property
		if i > 0 {
			// 1 to 5 dependencies
			// Reduce probability of dependencies for 100k nodes to keep connectivity reasonable
			if r.Float32() < 0.8 { // 80% chance of having dependencies
				numDeps := r.Intn(3) + 1 // 1 to 3 deps
				for range numDeps {
					// Pick any previous node.
					// Using Intn(i) creates a "random tree/graph" structure which typically has logarithmic depth.
					depIdx := r.Intn(i)
					depName := fmt.Sprintf("task-%d", depIdx)

					// Avoid duplicates
					exists := slices.Contains(tasks[i].Dependencies, depName)
					if !exists {
						tasks[i].Dependencies = append(tasks[i].Dependencies, depName)
					}
				}
			}
		}
	}

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: "benchmark-wf",
		},
		Status: wfv1.WorkflowStatus{
			Nodes: make(map[string]wfv1.NodeStatus),
		},
	}
	tmpl := &wfv1.Template{
		DAG: &wfv1.DAGTemplate{Tasks: tasks},
	}
	return wf, tmpl
}

// generateLinearChain creates a linear chain A→B→C→...→N where the first task
// has failed. This is the worst case for cascading omission: every downstream
// task must be evaluated and marked Omitted.
func generateLinearChain(n int) (*wfv1.Workflow, *wfv1.Template) {
	tasks := make([]wfv1.DAGTask, n)
	tasks[0] = wfv1.DAGTask{Name: "task-0"}
	for i := 1; i < n; i++ {
		tasks[i] = wfv1.DAGTask{
			Name:         fmt.Sprintf("task-%d", i),
			Dependencies: []string{fmt.Sprintf("task-%d", i-1)},
		}
	}

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "linear-chain"},
		Status:     wfv1.WorkflowStatus{Nodes: make(map[string]wfv1.NodeStatus)},
	}
	// Mark the first task as Failed to trigger cascading omission
	nodeID := wf.NodeID("dag.task-0")
	wf.Status.Nodes[nodeID] = wfv1.NodeStatus{
		ID:    nodeID,
		Name:  "dag.task-0",
		Phase: wfv1.NodeFailed,
		Type:  wfv1.NodeTypePod,
	}
	tmpl := &wfv1.Template{
		DAG: &wfv1.DAGTemplate{Tasks: tasks},
	}
	return wf, tmpl
}

// generateWideFanOut creates a single root with N-1 independent children.
// Tests the common fan-out pattern.
func generateWideFanOut(n int) (*wfv1.Workflow, *wfv1.Template) {
	tasks := make([]wfv1.DAGTask, n)
	tasks[0] = wfv1.DAGTask{Name: "root"}
	for i := 1; i < n; i++ {
		tasks[i] = wfv1.DAGTask{
			Name:         fmt.Sprintf("task-%d", i),
			Dependencies: []string{"root"},
		}
	}

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "fan-out"},
		Status:     wfv1.WorkflowStatus{Nodes: make(map[string]wfv1.NodeStatus)},
	}
	tmpl := &wfv1.Template{
		DAG: &wfv1.DAGTemplate{Tasks: tasks},
	}
	return wf, tmpl
}

func BenchmarkDAGEvaluator(b *testing.B) {
	sizes := []int{1000, 10000, 100000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("Random/Nodes-%d", n), func(b *testing.B) {
			b.StopTimer()
			wf, tmpl := generateRandomDAG(n)
			ctx := context.Background()
			evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_ = evaluator.EvaluateAll(ctx)
			}
		})
	}

	// Linear chain: worst case for cascading omission.
	// With old fixed-point: O(N²). With topological sort: O(N).
	chainSizes := []int{100, 1000, 5000}
	for _, n := range chainSizes {
		b.Run(fmt.Sprintf("LinearChain/Nodes-%d", n), func(b *testing.B) {
			b.StopTimer()
			wf, tmpl := generateLinearChain(n)
			ctx := context.Background()
			evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_ = evaluator.EvaluateAll(ctx)
			}
		})
	}

	// Fan-out: common pattern with many independent children.
	fanOutSizes := []int{1000, 10000}
	for _, n := range fanOutSizes {
		b.Run(fmt.Sprintf("FanOut/Nodes-%d", n), func(b *testing.B) {
			b.StopTimer()
			wf, tmpl := generateWideFanOut(n)
			ctx := context.Background()
			evaluator := NewDAGEvaluator(wf, tmpl, "", "dag")
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_ = evaluator.EvaluateAll(ctx)
			}
		})
	}

	// Construction cost: measures NewDAGEvaluator (topology parsing + topo sort)
	for _, n := range sizes {
		b.Run(fmt.Sprintf("Construction/Nodes-%d", n), func(b *testing.B) {
			b.StopTimer()
			wf, tmpl := generateRandomDAG(n)
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				_ = NewDAGEvaluator(wf, tmpl, "", "dag")
			}
		})
	}
}
