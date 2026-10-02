package hydrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/file"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// hydrationCostWorkflow models one Workflow, not one Workflow per retained Pod.
// The returned node map contains podNodes Pod nodes and one DAG root. Every
// synthetic Pod node has a 256-byte message and a 128-byte output parameter.
func hydrationCostWorkflow(podNodes int) *wfv1.Workflow {
	started := metav1.NewTime(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	wf := &wfv1.Workflow{
		TypeMeta: metav1.TypeMeta{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "hydration-cost", Namespace: "default",
			UID: "956ecb34-5a75-44cc-b030-689f44286477", ResourceVersion: "1",
			CreationTimestamp: started,
		},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "main",
			Templates: []wfv1.Template{
				{Name: "main", DAG: &wfv1.DAGTemplate{}},
				{Name: "worker", Container: &apiv1.Container{Image: "alpine:3.21", Command: []string{"true"}}},
			},
		},
		Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning, StartedAt: started, Nodes: wfv1.Nodes{}},
	}
	root := wfv1.NodeStatus{
		ID: wf.Name, Name: wf.Name, DisplayName: wf.Name, Type: wfv1.NodeTypeDAG,
		TemplateName: "main", TemplateScope: "local/" + wf.Name,
		Phase: wfv1.NodeRunning, StartedAt: started,
	}
	for i := range podNodes {
		task := fmt.Sprintf("task-%04d", i)
		name := wf.Name + "." + task
		id := wf.NodeID(name)
		message := fmt.Sprintf("node-%06d:", i)
		message += strings.Repeat("m", 256-len(message))
		parameter := fmt.Sprintf("output-%06d:", i)
		parameter += strings.Repeat("v", 128-len(parameter))
		wf.Spec.Templates[0].DAG.Tasks = append(wf.Spec.Templates[0].DAG.Tasks, wfv1.DAGTask{Name: task, Template: "worker"})
		root.Children = append(root.Children, id)
		wf.Status.Nodes[id] = wfv1.NodeStatus{
			ID: id, Name: name, DisplayName: task, Type: wfv1.NodeTypePod,
			TemplateName: "worker", TemplateScope: "local/" + wf.Name, BoundaryID: root.ID,
			Phase: wfv1.NodeRunning, StartedAt: started, Message: message,
			Outputs: &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "payload", Value: wfv1.AnyStringPtr(parameter)}}},
		}
	}
	wf.Status.Nodes[root.ID] = root
	return wf
}

// hydrationCostRepo measures the node JSON decoding performed by a successful
// offload Get. It does not model SQL, the network, connection pools or retries.
// Other repository methods deliberately remain unavailable to catch writes.
type hydrationCostRepo struct {
	sqldb.OffloadNodeStatusRepo
	data []byte
	gets int
}

func (r *hydrationCostRepo) Get(_ context.Context, uid, version string) (wfv1.Nodes, error) {
	if uid != "956ecb34-5a75-44cc-b030-689f44286477" || version != "cost-v1" {
		return nil, fmt.Errorf("unexpected offload identity %q/%q", uid, version)
	}
	r.gets++
	var nodes wfv1.Nodes
	err := json.Unmarshal(r.data, &nodes)
	return nodes, err
}

func hydrationCostRepresentation(ctx context.Context, tb testing.TB, wf *wfv1.Workflow, storage string) (*wfv1.Workflow, []byte) {
	tb.Helper()
	nodeJSON, err := json.Marshal(wf.Status.Nodes)
	if err != nil {
		tb.Fatal(err)
	}
	encoded := wf.DeepCopy()
	switch storage {
	case "raw":
	case "compressed":
		// Force a valid encoded representation. Dehydrate could silently turn a
		// small compressed fixture back into raw nodes during a benchmark.
		encoded.Status.Nodes = nil
		encoded.Status.CompressedNodes = file.CompressEncodeString(ctx, string(nodeJSON))
	case "offload":
		encoded.Status.Nodes = nil
		encoded.Status.OffloadNodeStatusVersion = "cost-v1"
	default:
		tb.Fatalf("unknown storage %q", storage)
	}
	return encoded, nodeJSON
}

// BenchmarkWorkflowHydrationCost separates the production Hydrate cost from
// Workflow JSON decoding plus Hydrate. repeats=16 represents repeated reads of
// the same Workflow node map, as can occur for 16 retained Pods, not 16 Workflows.
// Run with -benchmem. No API client or database latency is included.
func BenchmarkWorkflowHydrationCost(b *testing.B) {
	b.Setenv(file.CompressionAlgorithmEnvVarKey, file.GZipAlgorithm)
	b.Setenv(file.CompressionLevelEnvVarKey, "")
	ctx := logging.WithLogger(b.Context(), logging.NewSlogLoggerCustom(logging.Warn, logging.Text, io.Discard))
	for _, podNodes := range []int{32, 1024} {
		original := hydrationCostWorkflow(podNodes)
		for _, storage := range []string{"raw", "compressed", "offload"} {
			encoded, nodeJSON := hydrationCostRepresentation(ctx, b, original, storage)
			workflowJSON, err := json.Marshal(encoded)
			if err != nil {
				b.Fatal(err)
			}
			for _, decode := range []bool{false, true} {
				mode := "HydrateOnly"
				if decode {
					mode = "DecodeAndHydrate"
				}
				for _, repeats := range []int{1, 16} {
					b.Run(fmt.Sprintf("pods=%d/%s/%s/repeats=%d", podNodes, storage, mode, repeats), func(b *testing.B) {
						repo := &hydrationCostRepo{data: nodeJSON}
						h := New(repo)
						check := encoded.DeepCopy()
						if err := h.Hydrate(ctx, check); err != nil {
							b.Fatal(err)
						}
						checkJSON, err := json.Marshal(check.Status.Nodes)
						if err != nil {
							b.Fatal(err)
						}
						if !h.IsHydrated(check) || !bytes.Equal(checkJSON, nodeJSON) {
							b.Fatal("hydration changed or omitted the node result")
						}
						repo.gets = 0
						b.ReportAllocs()
						b.ResetTimer()
						for range b.N {
							for range repeats {
								wf := *encoded
								if decode {
									wf = wfv1.Workflow{}
									if err := json.Unmarshal(workflowJSON, &wf); err != nil {
										b.Fatal(err)
									}
								}
								if err := h.Hydrate(ctx, &wf); err != nil {
									b.Fatal(err)
								}
								if len(wf.Status.Nodes) != podNodes+1 {
									b.Fatal("hydration skipped nodes")
								}
							}
						}
						b.StopTimer()
						wantGets := 0
						if storage == "offload" {
							wantGets = b.N * repeats
						}
						if repo.gets != wantGets {
							b.Fatalf("logical repository Get count = %d, want %d", repo.gets, wantGets)
						}
						b.ReportMetric(float64(podNodes+1), "nodes/workflow")
						b.ReportMetric(float64(len(workflowJSON)), "workflow-bytes")
						b.ReportMetric(float64(len(nodeJSON)), "node-json-bytes")
						b.ReportMetric(float64(repeats), "hydrations/op")
						b.ReportMetric(float64(repo.gets)/float64(b.N), "repo-gets/op")
					})
				}
			}
		}
	}
}
