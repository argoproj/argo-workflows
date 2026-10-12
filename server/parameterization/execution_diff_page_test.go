package parameterization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func TestExecutionPagesCoverEveryLayerAndExcludeOnlyStatusVersion(t *testing.T) {
	changes := func(prefix string, count int) []p.Change {
		result := []p.Change{}
		for i := range count {
			result = append(result, p.Change{Op: "replace", Path: fmt.Sprintf("/%s/%d", prefix, i), Before: i, After: i + 1})
		}
		return result
	}
	full := &ExecutionComparison{Source: SourceExecution{Workflow: "original", UID: "uid"}, ResourceVersion: "1", ExecutionDigest: "target", Parameters: map[string]any{
		"left": "before", "right": "after", "changes": changes("values", 105), "modelChanges": changes("model", 1), "sourceChanges": changes("sources", 1),
	}, Layers: []ExecutionDiffLayer{
		{Name: "Form", Available: true, Changes: changes("form", 105)},
		{Name: "Compiled inputs", Available: true, Changes: changes("input", 5)},
		{Name: "Action binding", Available: false, Notice: "Historical snapshot missing", Changes: []p.Change{}},
	}}
	before := p.Digest(full)
	request := ExecutionDiffRequest{}
	paths := map[string]bool{}
	for pageNum := range 3 {
		page, err := executionDiffPage(full, request)
		require.NoError(t, err)
		require.Equal(t, 217, page.Pagination["total"])
		require.Equal(t, pageNum*100, page.Pagination["offset"])
		parameters := page.Parameters.(map[string]any)
		sections := [][]p.Change{parameters["changes"].([]p.Change), parameters["modelChanges"].([]p.Change), parameters["sourceChanges"].([]p.Change)}
		for _, layer := range page.Layers {
			sections = append(sections, layer.Changes)
		}
		count := 0
		for _, section := range sections {
			for _, change := range section {
				require.False(t, paths[change.Path])
				paths[change.Path] = true
				count++
			}
		}
		require.LessOrEqual(t, count, 100)
		require.Equal(t, 105, page.Layers[0].Total)
		require.False(t, page.Layers[2].Available)
		require.NotEmpty(t, page.Layers[2].Notice)
		request.Continue = page.Pagination["continue"].(string)
		request.Snapshot = page.Pagination["snapshot"].(string)
	}
	require.Empty(t, request.Continue)
	require.Len(t, paths, 217)
	require.Equal(t, before, p.Digest(full), "pagination does not mutate the source snapshot")
	full.ResourceVersion = "2"
	_, err := executionDiffPage(full, request)
	require.NoError(t, err, "native status-only changes do not change inputs")
	full.ExecutionDigest = "other-definition"
	_, err = executionDiffPage(full, request)
	require.ErrorContains(t, err, "comparison changed", "returning to first page must also check the same snapshot")
	request.Snapshot = ""
	request.Limit = 201
	_, err = executionDiffPage(full, request)
	require.Error(t, err)
}

func TestExecutionPageHTTPRechecksPermissionsAndLiveDefinitions(t *testing.T) {
	ctx, s, w, revision := previewFixture(t)
	h := Handler{Workflows: w, Gatekeeper: &testGatekeeper{client: s.Client}, Namespace: "lab"}
	preview, err := s.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	raw, err := h.submit(ctx, s, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "paged-execution"})
	require.NoError(t, err)
	wf := raw.(*wfv1.Workflow)
	wf.UID = "paged-uid"
	_, err = w.client.ArgoprojV1alpha1().Workflows("lab").Update(ctx, wf, metav1.UpdateOptions{})
	require.NoError(t, err)
	set, err := s.get(ctx, "parametersets", "sample")
	require.NoError(t, err)
	set, err = s.Save(ctx, "sample", SaveRequest{Template: "typed", Values: map[string]any{"count": 4, "enabled": false}, ResourceVersion: set.GetResourceVersion()})
	require.NoError(t, err)
	next, _, _ := unstructured.NestedString(set.Object, "spec", "revision")
	req := ExecutionDiffRequest{Source: SourceExecution{Workflow: wf.Name, UID: string(wf.UID)}, Revision: next, Limit: 1}
	post := func(code int) *ExecutionComparison {
		body, _ := json.Marshal(req)
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequestWithContext(ctx, http.MethodPost, Prefix+"lab/execution-diff", bytes.NewReader(body)))
		require.Equal(t, code, response.Code, response.Body.String())
		result := &ExecutionComparison{}
		if code == 200 {
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), result))
		}
		return result
	}
	first := post(200)
	req.Continue = first.Pagination["continue"].(string)
	req.Snapshot = first.Pagination["snapshot"].(string)
	require.NotEmpty(t, req.Continue)
	post(200)
	workflowTemplate, err := w.client.ArgoprojV1alpha1().WorkflowTemplates("lab").Get(ctx, "native", metav1.GetOptions{})
	require.NoError(t, err)
	workflowTemplate.Spec.Templates[0].Container.Image = "alpine:3.21"
	_, err = w.client.ArgoprojV1alpha1().WorkflowTemplates("lab").Update(ctx, workflowTemplate, metav1.UpdateOptions{})
	require.NoError(t, err)
	post(400)
	s.Client.(*dynamicfake.FakeDynamicClient).PrependReactor("get", "parameterrevisions", func(a clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: p.Group, Resource: "parameterrevisions"}, revision, nil)
	})
	post(403)
	require.Equal(t, 1, w.createCalls, "comparing must not submit or retry")
}
