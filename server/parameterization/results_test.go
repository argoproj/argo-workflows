package parameterization

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kt "k8s.io/client-go/testing"

	workflowapi "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	api "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

func resultFixture() (api.ActionResultDefinition, *wfv1.Template) {
	return api.ActionResultDefinition{Name: "report", ActionResultTemplateSpec: api.ActionResultTemplateSpec{Title: "Report", Source: api.ActionResultSource{Parameter: "report", Format: "json"}, Schema: api.JSONValue{Raw: json.RawMessage(`{"type":"object","required":["status"],"properties":{"status":{"type":"string"}},"additionalProperties":false}`)}, Views: []api.ResultView{{Type: "summary", Columns: []api.ResultColumn{{Title: "State", Path: "/status", Format: "status"}}}}}}, &wfv1.Template{Name: "emit", Container: &corev1.Container{Image: "alpine:3.22"}, Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "report"}, {Name: "resource-name"}, {Name: "resource-uid"}}}}
}

func TestResultContractValidationAndProjection(t *testing.T) {
	r, tmpl := resultFixture()
	require.NoError(t, validateActionResults([]api.ActionResultDefinition{r}, tmpl))
	v, err := normalizeResult(r, `{"status":"Ready"}`)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"status": "Ready"}, v)
	r.Mappings = []api.ResultFieldMapping{{From: "/payload/state", To: "status"}}
	v, err = normalizeResult(r, `{"payload":{"state":"Healthy"},"private":"must not escape"}`)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"status": "Healthy"}, v)
	_, err = normalizeResult(r, `{"payload":{}}`)
	require.ErrorContains(t, err, "ResultFieldMissing")
	_, err = normalizeResult(r, `{"payload":{"state":7}}`)
	require.ErrorContains(t, err, "ResultSchemaMismatch")
	for _, change := range []func(*api.ActionResultDefinition){
		func(x *api.ActionResultDefinition) { x.Source.Parameter = "missing" },
		func(x *api.ActionResultDefinition) { x.Source.Format = "javascript" },
		func(x *api.ActionResultDefinition) { x.Views[0].Columns[0].Path = "/~3invalid" },
		func(x *api.ActionResultDefinition) {
			x.Mappings = []api.ResultFieldMapping{{From: "", To: "same"}, {From: "/other", To: "same"}}
		},
		func(x *api.ActionResultDefinition) {
			x.Source.Parameter = ""
			x.Source.Resource = &api.ResultResourceSource{Version: "v1", Resource: "secrets", NameOutput: "resource-name", UIDOutput: "resource-uid", DocumentPath: "/data/value"}
		},
		func(x *api.ActionResultDefinition) {
			x.Schema.Raw = json.RawMessage(`{"$ref":"https://example.invalid/schema"}`)
		},
	} {
		x := r.DeepCopy()
		change(x)
		require.Error(t, validateActionResults([]api.ActionResultDefinition{*x}, tmpl))
	}
}

func TestResultBindingUsesFrozenCallOutputs(t *testing.T) {
	r, _ := resultFixture()
	call := api.ActionCall{CallPath: []string{"main", "east"}, ActionDefinition: "emit-v1", OutputMappings: []api.ActionParameterMapping{{From: "report", To: "receipt"}}}
	d := &api.ExecutionDefinitions{Actions: map[string]api.ActionDefinitionSpec{"emit-v1": {Results: []api.ActionResultDefinition{r}}}, ActionBinding: &api.WorkflowActionBindingSpec{Calls: []api.ActionCall{call}}}
	key := p.Digest("main/east")[:24]
	impl, err := resultDefinitions(d, wfv1.NodeStatus{TemplateName: "action-" + key})
	require.NoError(t, err)
	require.Equal(t, "report", impl[0].Source.Parameter)
	wrapper, err := resultDefinitions(d, wfv1.NodeStatus{TemplateName: "call-" + key})
	require.NoError(t, err)
	require.Equal(t, "receipt", wrapper[0].Source.Parameter)
	missing, err := resultDefinitions(d, wfv1.NodeStatus{TemplateName: "different"})
	require.NoError(t, err)
	require.Empty(t, missing)
	require.Equal(t, "report", d.Actions["emit-v1"].Results[0].Source.Parameter)
}

type resultWorkflowService struct {
	workflowapi.WorkflowServiceServer
	workflow *wfv1.Workflow
}

func (s resultWorkflowService) GetWorkflow(context.Context, *workflowapi.WorkflowGetRequest) (*wfv1.Workflow, error) {
	return s.workflow.DeepCopy(), nil
}

func TestNodeResultsReadOnlySnapshotAndControllerObservation(t *testing.T) {
	ctx := t.Context()
	s := testStore()
	r, tmpl := resultFixture()
	live := *r.DeepCopy()
	live.Name = "live"
	live.Source.Parameter = ""
	live.Source.Resource = &api.ResultResourceSource{Version: "v1", Resource: "configmaps", NameOutput: "resource-name", UIDOutput: "resource-uid", DocumentPath: "/data/report", ObservedAtPath: "/data/observedAt", MaxAgeSeconds: 30}
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "result-demo", Namespace: s.Namespace, UID: types.UID("workflow-id"), Annotations: map[string]string{p.Group + "/submission": "submission", p.Group + "/revision": "revision"}}, Spec: wfv1.WorkflowSpec{Templates: []wfv1.Template{*tmpl}}, Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded, Nodes: wfv1.Nodes{"node": {ID: "node", Name: "result-demo.emit", TemplateName: "emit", Phase: wfv1.NodeSucceeded, StartedAt: metav1.Now(), Outputs: &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "report", Value: wfv1.AnyStringPtr(`{"status":"Completed"}`)}, {Name: "resource-name", Value: wfv1.AnyStringPtr("report")}, {Name: "resource-uid", Value: wfv1.AnyStringPtr("report-id")}}}}}}}
	defs := api.ExecutionDefinitions{ActionBinding: &api.WorkflowActionBindingSpec{Calls: []api.ActionCall{{Template: "emit", ActionDefinition: "action"}}}, Actions: map[string]api.ActionDefinitionSpec{"action": {Results: []api.ActionResultDefinition{r, live}}}}
	_, err := s.resource("workflowsubmissions").Create(ctx, object("WorkflowSubmission", "submission", s.Namespace, map[string]any{"workflow": wf, "revision": "revision", "definitions": defs}), metav1.CreateOptions{})
	require.NoError(t, err)
	resource := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	cm := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "report", "namespace": s.Namespace, "uid": "report-id"}, "data": map[string]any{"report": `{"status":"Healthy"}`, "observedAt": time.Now().Add(-time.Minute).Format(time.RFC3339)}}}
	cm.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow", Name: wf.Name, UID: wf.UID}})
	_, err = s.Client.Resource(resource).Namespace(s.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	require.NoError(t, err)
	h := Handler{Workflows: resultWorkflowService{workflow: wf}}
	client := s.Client.(*dynamicfake.FakeDynamicClient)
	client.ClearActions()
	for i := 0; i < 3; i++ {
		result, readErr := h.nodeResults(ctx, s, wf.Name, string(wf.UID), "node", "")
		require.NoError(t, readErr)
		require.Len(t, result.Items, 2)
		require.True(t, result.Items[0].Available)
		require.Equal(t, "ExecutionSnapshot", result.Items[0].Scope)
		require.Equal(t, "Completed", result.Items[0].Value.(map[string]any)["status"])
		require.True(t, result.Items[1].Stale)
		require.Equal(t, "LiveObservation", result.Items[1].Scope)
	}
	for _, a := range client.Actions() {
		require.Equal(t, "get", a.GetVerb(), "refresh never dispatches or mutates anything")
	}
	require.Equal(t, wfv1.WorkflowSucceeded, wf.Status.Phase)
	_, err = h.nodeResults(ctx, s, wf.Name, "replaced", "node", "")
	require.ErrorContains(t, err, "identity")
	_, err = h.nodeResults(ctx, s, wf.Name, string(wf.UID), "another-node", "")
	require.ErrorContains(t, err, "does not belong")
	cm.SetUID("changed")
	_, err = s.Client.Resource(resource).Namespace(s.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	require.NoError(t, err)
	result, err := h.nodeResults(ctx, s, wf.Name, string(wf.UID), "node", "")
	require.NoError(t, err)
	require.Equal(t, "ResourceIdentityChanged", result.Items[1].Error)
	require.Nil(t, result.Items[1].Value)
	client.PrependReactor("get", "configmaps", func(kt.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(resource.GroupResource(), "report", nil)
	})
	result, err = h.nodeResults(ctx, s, wf.Name, string(wf.UID), "node", "")
	require.NoError(t, err)
	require.Equal(t, "AccessDenied", result.Items[1].Error)
	require.Nil(t, result.Items[1].Value)
}

func TestReusableResultsBindIndependentlyAtEachCall(t *testing.T) {
	r, tmpl := resultFixture()
	library := r.ActionResultTemplateSpec.DeepCopy()
	library.Source.Parameter = "payload"
	lookup := func(name string) (*api.ActionResultTemplateSpec, error) {
		require.Equal(t, "public-v1", name)
		return library, nil
	}
	call := api.ActionCall{CallPath: []string{"main", "east"}, ActionDefinition: "emit", ResultBindings: []api.ActionResultBinding{{Name: "east-report", ResultTemplate: "public-v1", OutputMappings: []api.ActionParameterMapping{{From: "report", To: "payload"}}}}}
	bound, err := bindActionResults(call, tmpl, lookup)
	require.NoError(t, err)
	require.Equal(t, "report", bound[0].Source.Parameter)
	require.Equal(t, "payload", library.Source.Parameter, "the reusable definition is not rewritten")
	other := *call.DeepCopy()
	other.CallPath[1] = "west"
	other.ResultBindings[0].Name = "west-report"
	second, err := bindActionResults(other, tmpl, lookup)
	require.NoError(t, err)
	defs := &api.ExecutionDefinitions{ActionBinding: &api.WorkflowActionBindingSpec{Calls: []api.ActionCall{call, other}}, Actions: map[string]api.ActionDefinitionSpec{"emit": {}}, CallResults: map[string][]api.ActionResultDefinition{resultCallKey(call): bound, resultCallKey(other): second}}
	for _, c := range []api.ActionCall{call, other} {
		results, readErr := resultDefinitions(defs, wfv1.NodeStatus{TemplateName: "action-" + p.Digest(strings.Join(c.CallPath, "/"))[:24]})
		require.NoError(t, readErr)
		require.Equal(t, c.ResultBindings[0].Name, results[0].Name)
	}
	delete(defs.CallResults, resultCallKey(call))
	_, err = resultDefinitions(defs, wfv1.NodeStatus{TemplateName: "action-" + p.Digest("main/east")[:24]})
	require.ErrorContains(t, err, "frozen result binding")
	for _, mapping := range [][]api.ActionParameterMapping{
		{{From: "absent", To: "payload"}}, {{From: "report", To: "unknown"}}, {{From: "report", To: "payload"}, {From: "report", To: "payload"}},
	} {
		bad := *call.DeepCopy()
		bad.ResultBindings[0].OutputMappings = mapping
		_, err = bindActionResults(bad, tmpl, lookup)
		require.Error(t, err)
	}
	status := EvaluateDefinition(object("ActionResultTemplate", "public-v1", "lab", library), func(DefinitionKey) (*unstructured.Unstructured, error) {
		t.Fatal("no external reference required")
		return nil, nil
	})
	require.Empty(t, status.Errors)
}

func TestResultContractsSurviveUnrelatedPartialRetry(t *testing.T) {
	s := testStore()
	ctx := t.Context()
	r, tmpl := resultFixture()
	old := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: s.Namespace, UID: "same-workflow", Annotations: map[string]string{p.Group + "/revision": "old", p.Group + "/submission": "original"}}, Spec: wfv1.WorkflowSpec{Templates: []wfv1.Template{*tmpl}}, Status: wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"retained": {ID: "retained", TemplateName: "emit", Phase: wfv1.NodeSucceeded, StartedAt: metav1.Now(), FinishedAt: metav1.Now(), Outputs: &wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "report", Value: wfv1.AnyStringPtr(`{"status":"original"}`)}}}}}}}
	oldDefs := api.ExecutionDefinitions{Actions: map[string]api.ActionDefinitionSpec{"emit": {Results: []api.ActionResultDefinition{r}}}, ActionBinding: &api.WorkflowActionBindingSpec{Calls: []api.ActionCall{{Template: "emit", ActionDefinition: "emit"}}}}
	_, err := s.resource("workflowsubmissions").Create(ctx, object("WorkflowSubmission", "original", s.Namespace, api.WorkflowSubmissionSpec{Workflow: *old, Revision: "old", Definitions: &oldDefs}), metav1.CreateOptions{})
	require.NoError(t, err)
	next := old.DeepCopy()
	next.Annotations[p.Group+"/retry"] = "retry-one"
	next.Annotations[p.Group+"/revision"] = "new"
	newDefs := oldDefs.DeepCopy()
	newDefs.Actions["emit"] = api.ActionDefinitionSpec{Results: []api.ActionResultDefinition{*r.DeepCopy()}}
	newDefs.Actions["emit"].Results[0].Title = "New contract"
	_, err = s.resource("parameterretries").Create(ctx, object("ParameterRetry", "retry-one", s.Namespace, api.ParameterRetrySpec{Workflow: next.Name, WorkflowUID: string(next.UID), Revision: "new", PreviousWorkflow: old, Definitions: newDefs}), metav1.CreateOptions{})
	require.NoError(t, err)
	h := Handler{Workflows: resultWorkflowService{workflow: next}}
	got, err := h.nodeResults(ctx, s, next.Name, string(next.UID), "retained", "")
	require.NoError(t, err)
	require.Equal(t, "Report", got.Items[0].Title)
	previous, err := h.nodeResults(ctx, s, next.Name, string(next.UID), "retained", "retry-one")
	require.NoError(t, err)
	require.Equal(t, &NodeResults{Workflow: previous.Workflow, UID: previous.UID, NodeID: previous.NodeID, Invocation: previous.Invocation, ContractDigest: previous.ContractDigest, Items: previous.Items}, got)
	node := next.Status.Nodes["retained"]
	node.StartedAt = metav1.NewTime(node.StartedAt.Add(time.Minute))
	node.Outputs.Parameters[0].Value = wfv1.AnyStringPtr(`{"status":"new attempt"}`)
	next.Status.Nodes["retained"] = node
	h.Workflows = resultWorkflowService{workflow: next}
	current, err := h.nodeResults(ctx, s, next.Name, string(next.UID), "retained", "")
	require.NoError(t, err)
	require.Equal(t, "New contract", current.Items[0].Title)
	require.NotEqual(t, got.Invocation, current.Invocation)
}

// A producer can allocate a public observation object before dispatch. The
// immutable input identity makes it readable while the action has no outputs.
func TestLiveResultIdentityFromInvocationInputs(t *testing.T) {
	ctx := t.Context()
	s := testStore()
	r, tmpl := resultFixture()
	r.Source.Parameter = ""
	r.Source.Resource = &api.ResultResourceSource{Version: "v1", Resource: "configmaps", NameInput: "result-name", UIDInput: "result-uid", DocumentPath: "/data/report"}
	tmpl.Inputs.Parameters = []wfv1.Parameter{{Name: "result-name"}, {Name: "result-uid"}}
	require.NoError(t, validateActionResults([]api.ActionResultDefinition{r}, tmpl))
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "live", Namespace: s.Namespace, UID: "wf"}}
	node := wfv1.NodeStatus{Phase: wfv1.NodeRunning, Inputs: &wfv1.Inputs{Parameters: []wfv1.Parameter{{Name: "result-name", Value: wfv1.AnyStringPtr("report")}, {Name: "result-uid", Value: wfv1.AnyStringPtr("report-uid")}}}}
	cm := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap", "metadata": map[string]any{"name": "report", "namespace": s.Namespace, "uid": "report-uid"}, "data": map[string]any{"report": `{"status":"Running"}`}}}
	cm.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "argoproj.io/v1alpha1", Kind: "Workflow", Name: wf.Name, UID: wf.UID}})
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	_, err := s.Client.Resource(gvr).Namespace(s.Namespace).Create(ctx, cm, metav1.CreateOptions{})
	require.NoError(t, err)
	for _, phase := range []wfv1.NodePhase{wfv1.NodeRunning, wfv1.NodeFailed, wfv1.NodeSucceeded} {
		node.Phase = phase
		out := s.readResult(ctx, wf, node, r)
		require.True(t, out.Available, out.Error)
		require.Equal(t, map[string]any{"status": "Running"}, out.Value)
	}
	for _, change := range []func(*api.ResultResourceSource){
		func(r *api.ResultResourceSource) { r.UIDInput = "" }, func(r *api.ResultResourceSource) { r.UIDInput = r.NameInput },
		func(r *api.ResultResourceSource) { r.UIDInput = "absent" }, func(r *api.ResultResourceSource) { r.NameOutput = "resource-name"; r.UIDOutput = "resource-uid" },
	} {
		bad := r.DeepCopy()
		change(bad.Source.Resource)
		require.Error(t, validateActionResults([]api.ActionResultDefinition{*bad}, tmpl))
	}
	other := wf.DeepCopy()
	other.UID = "different"
	require.Equal(t, "ResourceOwnershipUnverified", s.readResult(ctx, other, node, r).Error)
	missing := node
	missing.Inputs = nil
	require.Equal(t, "ResourceIdentityPending", s.readResult(ctx, wf, missing, r).Error)
	node.Inputs.Parameters[1].Value = wfv1.AnyStringPtr("replacement")
	require.Equal(t, "ResourceIdentityChanged", s.readResult(ctx, wf, node, r).Error)

	library := r.ActionResultTemplateSpec.DeepCopy()
	library.Source.Resource.NameInput = "object"
	library.Source.Resource.UIDInput = "identity"
	call := api.ActionCall{CallPath: []string{"main", "call"}, ActionDefinition: "observe", ResultBindings: []api.ActionResultBinding{{Name: "live", ResultTemplate: "observe", InputMappings: []api.ActionParameterMapping{{From: "result-name", To: "object"}, {From: "result-uid", To: "identity"}}}}}
	bound, err := bindActionResults(call, tmpl, func(string) (*api.ActionResultTemplateSpec, error) { return library, nil })
	require.NoError(t, err)
	require.Equal(t, "result-name", bound[0].Source.Resource.NameInput)
	call.InputMappings = []api.ActionParameterMapping{{From: "original-name", To: "result-name"}, {From: "original-uid", To: "result-uid"}}
	defs := &api.ExecutionDefinitions{Actions: map[string]api.ActionDefinitionSpec{"observe": {}}, ActionBinding: &api.WorkflowActionBindingSpec{Calls: []api.ActionCall{call}}, CallResults: map[string][]api.ActionResultDefinition{resultCallKey(call): bound}}
	results, err := resultDefinitions(defs, wfv1.NodeStatus{TemplateName: "call-" + p.Digest("main/call")[:24]})
	require.NoError(t, err)
	require.Equal(t, "original-name", results[0].Source.Resource.NameInput)
	status := EvaluateDefinition(object("ActionResultTemplate", "observe", s.Namespace, library), func(DefinitionKey) (*unstructured.Unstructured, error) { t.Fatal("unexpected lookup"); return nil, nil })
	require.Empty(t, status.Errors)
	call.ResultBindings[0].InputMappings = append(call.ResultBindings[0].InputMappings, api.ActionParameterMapping{From: "result-name", To: "object"})
	_, err = bindActionResults(call, tmpl, func(string) (*api.ActionResultTemplateSpec, error) { return library, nil })
	require.Error(t, err)
}
