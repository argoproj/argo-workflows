package executor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestManifestDocCount(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		want     int
	}{
		{"single", "apiVersion: v1\nkind: ConfigMap", 1},
		{"leading-separator", "---\napiVersion: v1\nkind: ConfigMap", 1},
		{"trailing-separator", "apiVersion: v1\nkind: ConfigMap\n---\n", 1},
		{"multi", "kind: ConfigMap\n---\nkind: Secret", 2},
		{"multi-with-trailing", "kind: ConfigMap\n---\nkind: Secret\n---\n", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, common.ManifestDocCount([]byte(tt.manifest)))
		})
	}
}

func TestWithAgentMetadata(t *testing.T) {
	resource := &wfv1.ResourceTemplate{SuccessCondition: "status.phase == Running"}

	t.Run("single doc gets label and annotations", func(t *testing.T) {
		manifest := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm"
		out, obj, err := withAgentMetadata([]byte(manifest), "wf-1", "uid-1", "node-1", resource)
		require.NoError(t, err)
		assert.Equal(t, "cm", obj.GetName())
		assert.Equal(t, "uid-1", obj.GetLabels()[common.LabelKeyAgentResource])
		assert.Equal(t, "node-1", obj.GetAnnotations()[common.AnnotationKeyNodeID])
		assert.Equal(t, "status.phase == Running", obj.GetAnnotations()[common.AnnotationKeySuccessCondition])
		// no ownerReference unless setOwnerReference is requested
		assert.Empty(t, obj.GetOwnerReferences())

		// the returned manifest round-trips the same metadata
		reparsed := map[string]any{}
		require.NoError(t, yaml.Unmarshal(out, &reparsed))
		assert.Equal(t, "cm", reparsed["metadata"].(map[string]any)["name"])
	})

	t.Run("setOwnerReference injects the workflow owner ref", func(t *testing.T) {
		manifest := "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm"
		ownedResource := &wfv1.ResourceTemplate{SetOwnerReference: true}
		_, obj, err := withAgentMetadata([]byte(manifest), "wf-1", "uid-1", "node-1", ownedResource)
		require.NoError(t, err)
		refs := obj.GetOwnerReferences()
		require.Len(t, refs, 1)
		assert.Equal(t, "wf-1", refs[0].Name)
		assert.Equal(t, "uid-1", string(refs[0].UID))
		assert.Equal(t, "Workflow", refs[0].Kind)
		require.NotNil(t, refs[0].Controller)
		assert.True(t, *refs[0].Controller)
	})

	t.Run("multi doc is rejected, not silently partly applied", func(t *testing.T) {
		manifest := "kind: ConfigMap\nmetadata:\n  name: cm\n---\nkind: Secret\nmetadata:\n  name: s"
		_, _, err := withAgentMetadata([]byte(manifest), "wf-1", "uid-1", "node-1", resource)
		require.Error(t, err)
	})
}

func TestParseConditions(t *testing.T) {
	// whitespace-only conditions parse to zero requirements (treated as "nothing to wait for"),
	// matching the per-pod WaitResource, rather than erroring or hanging.
	rc, err := parseConditions(" ", "")
	require.NoError(t, err)
	assert.Empty(t, rc.successReqs)
	assert.Empty(t, rc.failReqs)

	rc, err = parseConditions("status.phase == Running", "")
	require.NoError(t, err)
	assert.Len(t, rc.successReqs, 1)

	_, err = parseConditions("!!!bad", "")
	require.Error(t, err)
}

func TestEnsureInformerRBAC(t *testing.T) {
	// When the agent's service account cannot list/watch a GVR, ensureInformer must surface the
	// error instead of starting an informer that silently retries forever and hangs the node.
	ctx := logging.TestContext(t.Context())
	gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}
	client := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "WidgetList"})
	client.PrependReactor("list", "*", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "widgets"}, "", errors.New("forbidden"))
	})
	rae := &ResourceAgentExecutor{
		WorkflowUID:   "uid-1",
		DynamicClient: client,
		informers:     map[informerKey]cache.SharedIndexInformer{},
	}

	err := rae.ensureInformer(ctx, gvr, "default")
	require.ErrorContains(t, err, "cannot watch")
	assert.Empty(t, rae.informers, "no informer should be registered when the watch is forbidden")
}

func TestSaveResourceParameters(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "cm"},
		"status":   map[string]any{"phase": "Running"},
	}}

	t.Run("delete falls back to default", func(t *testing.T) {
		tmpl := &wfv1.Template{
			Resource: &wfv1.ResourceTemplate{Action: "delete"},
			Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{
				Name:      "p",
				ValueFrom: &wfv1.ValueFrom{Default: wfv1.AnyStringPtr("fallback")},
			}}},
		}
		outputs, err := saveResourceParameters(ctx, tmpl, obj)
		require.NoError(t, err)
		require.Len(t, outputs.Parameters, 1)
		assert.Equal(t, "fallback", outputs.Parameters[0].Value.String())
	})

	t.Run("jsonPath reads the object", func(t *testing.T) {
		tmpl := &wfv1.Template{
			Resource: &wfv1.ResourceTemplate{Action: "create"},
			Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{
				Name:      "phase",
				ValueFrom: &wfv1.ValueFrom{JSONPath: "{.status.phase}"},
			}}},
		}
		outputs, err := saveResourceParameters(ctx, tmpl, obj)
		require.NoError(t, err)
		require.Len(t, outputs.Parameters, 1)
		assert.Equal(t, "Running", outputs.Parameters[0].Value.String())
	})
}

func objWithSuccessCondition(nodeID, phase string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"name": "cm",
			"annotations": map[string]any{
				common.AnnotationKeyNodeID:           nodeID,
				common.AnnotationKeySuccessCondition: "status.phase == Running",
			},
		},
		"status": map[string]any{"phase": phase},
	}}
}

func newTestResourceAgent() *ResourceAgentExecutor {
	return &ResourceAgentExecutor{
		tasks:    map[string]*wfv1.Template{},
		pending:  map[string]wfv1.NodeResult{},
		reported: map[string]bool{},
	}
}

func TestProcessEventSuccessThenDelete(t *testing.T) {
	// A resource that met its success condition and is then deleted must report Succeeded. The
	// delete event carries the last-known object (which still satisfies the condition), so the
	// verdict must come from that object, not from a store the deletion has already emptied.
	ctx := logging.TestContext(t.Context())
	rae := newTestResourceAgent()
	rae.tasks["node-1"] = &wfv1.Template{Resource: &wfv1.ResourceTemplate{}}
	rae.processEvent(ctx, resourceEvent{obj: objWithSuccessCondition("node-1", "Running"), deleted: true, nodeID: "node-1"})
	rae.resultsMutex.Lock()
	defer rae.resultsMutex.Unlock()
	require.Contains(t, rae.pending, "node-1")
	assert.Equal(t, wfv1.NodeSucceeded, rae.pending["node-1"].Phase)
}

func TestProcessEventDeletedBeforeSuccess(t *testing.T) {
	// A resource deleted before meeting its success condition fails, as before.
	ctx := logging.TestContext(t.Context())
	rae := newTestResourceAgent()
	rae.tasks["node-1"] = &wfv1.Template{Resource: &wfv1.ResourceTemplate{}}
	rae.processEvent(ctx, resourceEvent{obj: objWithSuccessCondition("node-1", "Pending"), deleted: true, nodeID: "node-1"})
	rae.resultsMutex.Lock()
	defer rae.resultsMutex.Unlock()
	require.Contains(t, rae.pending, "node-1")
	assert.Equal(t, wfv1.NodeFailed, rae.pending["node-1"].Phase)
}

func TestProcessEventStaleNodeSkipped(t *testing.T) {
	// A watch event for a node this agent is not tracking (no registered template) — e.g. a
	// completed-and-pruned node whose object still exists and is re-listed after a restart — must be
	// skipped, so its nil-output result can't clobber the node's recorded outputs in the controller.
	ctx := logging.TestContext(t.Context())
	rae := newTestResourceAgent() // no template registered for node-1
	rae.processEvent(ctx, resourceEvent{obj: objWithSuccessCondition("node-1", "Running"), nodeID: "node-1"})
	rae.resultsMutex.Lock()
	defer rae.resultsMutex.Unlock()
	assert.NotContains(t, rae.pending, "node-1", "a stale/untracked node must not be reported")
}
