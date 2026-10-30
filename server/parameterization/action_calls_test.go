package parameterization

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	parameterapi "github.com/argoproj/argo-workflows/v4/pkg/apis/parameterization/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func actionCallFixture() (*wfv1.Workflow, ActionCall, map[string]*wfv1.Template) {
	baseline := wfv1.Template{Name: "old", Inputs: wfv1.Inputs{Parameters: []wfv1.Parameter{{Name: "message"}}}, Outputs: wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "reply"}}}, Container: &corev1.Container{Image: "alpine"}}
	inner := wfv1.Template{Name: "shared", Metadata: wfv1.Metadata{Annotations: map[string]string{bindableCallsAnnotation: `["emit"]`}}, DAG: &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{{Name: "emit", Template: "old", When: "{{inputs.parameters.message}} != skip", Arguments: wfv1.Arguments{Parameters: []wfv1.Parameter{{Name: "message", Value: wfv1.AnyStringPtr("{{inputs.parameters.message}}")}}}}}}}
	root := wfv1.Template{Name: "main", DAG: &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{{Name: "east", Template: "shared"}, {Name: "west", Template: "shared", Depends: "east.Succeeded"}}}}
	wf := &wfv1.Workflow{Spec: wfv1.WorkflowSpec{Entrypoint: "main", Templates: []wfv1.Template{root, inner, baseline}}}
	call := ActionCall{CallPath: []string{"main", "east", "emit"}, ActionDefinition: "echo", InputMappings: []parameterapi.ActionParameterMapping{{From: "message", To: "value"}}, OutputMappings: []parameterapi.ActionParameterMapping{{From: "answer", To: "reply"}}}
	impl := baseline.DeepCopy()
	impl.Name = "impl"
	impl.Inputs.Parameters[0].Name = "value"
	impl.Outputs.Parameters[0].Name = "answer"
	return wf, call, map[string]*wfv1.Template{"echo": impl}
}

func TestActionCallPathClonesOnlySelectedInvocation(t *testing.T) {
	wf, call, impls := actionCallFixture()
	original := wf.DeepCopy()
	require.NoError(t, compileActionCalls(wf, []ActionCall{call}, impls))
	require.Equal(t, original.Spec.Templates, wf.Spec.Templates[:3], "shared original templates are immutable")
	main := definitionTemplate(&wf.Spec, wf.Spec.Entrypoint)
	require.Equal(t, "shared", main.DAG.Tasks[1].Template)
	require.Equal(t, "east.Succeeded", main.DAG.Tasks[1].Depends)
	east := definitionTemplate(&wf.Spec, main.DAG.Tasks[0].Template)
	require.Equal(t, original.Spec.Templates[1].DAG.Tasks[0].When, east.DAG.Tasks[0].When)
	require.Equal(t, original.Spec.Templates[1].DAG.Tasks[0].Arguments, east.DAG.Tasks[0].Arguments)
	wrapper := definitionTemplate(&wf.Spec, east.DAG.Tasks[0].Template)
	require.Equal(t, "value", wrapper.DAG.Tasks[0].Arguments.Parameters[0].Name)
	require.Equal(t, "{{inputs.parameters.message}}", wrapper.DAG.Tasks[0].Arguments.Parameters[0].Value.String())
	require.Equal(t, "reply", wrapper.Outputs.Parameters[0].Name)
	require.Equal(t, "{{tasks.invoke.outputs.parameters.answer}}", wrapper.Outputs.Parameters[0].ValueFrom.Parameter)
	again := original.DeepCopy()
	require.NoError(t, compileActionCalls(again, []ActionCall{call}, impls))
	require.Equal(t, wf.Spec, again.Spec, "stable names permit native node matching on partial retry")
	// A second selected invocation is independent even when both use shared.
	other := call
	other.CallPath = []string{"main", "west", "emit"}
	require.NoError(t, compileActionCalls(original, []ActionCall{call, other}, impls))
	root := definitionTemplate(&original.Spec, original.Spec.Entrypoint)
	require.NotEqual(t, root.DAG.Tasks[0].Template, root.DAG.Tasks[1].Template)
}

func TestActionCallPathStepsAndLoopSite(t *testing.T) {
	wf, call, impls := actionCallFixture()
	shared := &wf.Spec.Templates[1]
	shared.DAG = nil
	shared.Steps = []wfv1.ParallelSteps{{Steps: []wfv1.WorkflowStep{{Name: "emit", Template: "old", WithParam: "{{inputs.parameters.items}}"}}}}
	require.NoError(t, compileActionCalls(wf, []ActionCall{call}, impls))
	main := definitionTemplate(&wf.Spec, wf.Spec.Entrypoint)
	clone := definitionTemplate(&wf.Spec, main.DAG.Tasks[0].Template)
	require.Equal(t, "{{inputs.parameters.items}}", clone.Steps[0].Steps[0].WithParam)
	require.Equal(t, "emit", clone.Steps[0].Steps[0].Name)
}

func TestActionCallPathRejectsAmbiguityAndUnapprovedBindings(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*wfv1.Workflow, *ActionCall, map[string]*wfv1.Template)
		want   string
	}{
		{"absent", func(w *wfv1.Workflow, c *ActionCall, _ map[string]*wfv1.Template) { c.CallPath[2] = "missing" }, "one named"},
		{"entrypoint", func(w *wfv1.Workflow, c *ActionCall, _ map[string]*wfv1.Template) { c.CallPath[0] = "shared" }, "entrypoint"},
		{"not opted in", func(w *wfv1.Workflow, c *ActionCall, _ map[string]*wfv1.Template) {
			w.Spec.Templates[1].Metadata.Annotations = nil
		}, "author-approved"},
		{"duplicate mapping", func(w *wfv1.Workflow, c *ActionCall, _ map[string]*wfv1.Template) {
			c.InputMappings = append(c.InputMappings, c.InputMappings[0])
		}, "duplicated"},
		{"missing mapping", func(w *wfv1.Workflow, c *ActionCall, _ map[string]*wfv1.Template) { c.OutputMappings = nil }, "unknown"},
		{"artifact", func(w *wfv1.Workflow, c *ActionCall, _ map[string]*wfv1.Template) {
			w.Spec.Templates[2].Inputs.Artifacts = []wfv1.Artifact{{Name: "data"}}
		}, "input artifact mapping"},
		{"ancestor loop", func(w *wfv1.Workflow, c *ActionCall, _ map[string]*wfv1.Template) {
			w.Spec.Templates[0].DAG.Tasks[0].WithParam = "[1,2]"
		}, "descend through a loop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wf, call, impl := actionCallFixture()
			tc.change(wf, &call, impl)
			require.ErrorContains(t, compileActionCalls(wf, []ActionCall{call}, impl), tc.want)
		})
	}
	_, call, _ := actionCallFixture()
	require.ErrorContains(t, checkActionCalls([]ActionCall{call, call}), "overlapping")
	parent := call
	parent.CallPath = parent.CallPath[:2]
	require.ErrorContains(t, checkActionCalls([]ActionCall{call, parent}), "overlapping")
	require.ErrorContains(t, checkActionCalls([]ActionCall{call, {Template: "old"}}), "cannot be mixed")
}

func TestActionCallPathReferencedAncestorStaticValidation(t *testing.T) {
	wf, call, impls := actionCallFixture()
	library := wf.Spec.DeepCopy()
	wf.Spec.Templates[0].DAG.Tasks[0].Template = ""
	wf.Spec.Templates[0].DAG.Tasks[0].TemplateRef = &wfv1.TemplateRef{Name: "library", Template: "shared", ClusterScope: true}
	baseline, err := actionCallBaseline(&wf.Spec, call.CallPath, func(ref *wfv1.TemplateRef) (*wfv1.WorkflowSpec, error) {
		require.True(t, ref.ClusterScope)
		return library, nil
	})
	require.NoError(t, err)
	require.Equal(t, "old", baseline.Name)
	impl := impls["echo"]
	raw, _ := json.Marshal(impl)
	require.NotEmpty(t, raw)
	_, err = actionAdapter(baseline, call, impl, "adapter")
	require.NoError(t, err)
}

func TestActionArtifactMappings(t *testing.T) {
	wf, call, impls := actionCallFixture()
	baseline := &wf.Spec.Templates[2]
	impl := impls["echo"]
	baseline.Inputs.Artifacts = []wfv1.Artifact{{Name: "source", Path: "/old/input"}}
	baseline.Outputs.Artifacts = []wfv1.Artifact{{Name: "report", Path: "/old/output", GlobalName: "public-report"}}
	impl.Inputs.Artifacts = []wfv1.Artifact{{Name: "data", Path: "/new/input"}}
	impl.Outputs.Artifacts = []wfv1.Artifact{{Name: "result", Path: "/new/output"}}
	call.InputArtifactMappings = []parameterapi.ActionParameterMapping{{From: "source", To: "data"}}
	call.OutputArtifactMappings = []parameterapi.ActionParameterMapping{{From: "result", To: "report"}}
	adapter, err := actionAdapter(baseline, call, impl, "adapter")
	require.NoError(t, err)
	require.Empty(t, adapter.Inputs.Artifacts[0].Path)
	require.Equal(t, "{{inputs.artifacts.source}}", adapter.DAG.Tasks[0].Arguments.Artifacts[0].From)
	require.Equal(t, "{{tasks.invoke.outputs.artifacts.result}}", adapter.Outputs.Artifacts[0].From)
	require.Equal(t, "public-report", adapter.Outputs.Artifacts[0].GlobalName)
	require.Empty(t, adapter.Outputs.Artifacts[0].Path)
	require.Equal(t, "/new/output", impl.Outputs.Artifacts[0].Path)
	baseline.Inputs.Artifacts[0].Optional = true
	_, err = actionAdapter(baseline, call, impl, "adapter")
	require.ErrorContains(t, err, "optional input")
	impl.Inputs.Artifacts[0].Optional = true
	impl.Outputs.Artifacts[0].Optional = true
	_, err = actionAdapter(baseline, call, impl, "adapter")
	require.ErrorContains(t, err, "optional action output")
	baseline.Outputs.Artifacts[0].Optional = true
	_, err = actionAdapter(baseline, call, impl, "adapter")
	require.NoError(t, err)
	impl.Outputs.Artifacts[0].GlobalName = "hidden-export"
	_, err = actionAdapter(baseline, call, impl, "adapter")
	require.ErrorContains(t, err, "global exports")
	call.OutputArtifactMappings = nil
	_, err = actionAdapter(baseline, call, impl, "adapter")
	require.ErrorContains(t, err, "output artifact mapping")
}
