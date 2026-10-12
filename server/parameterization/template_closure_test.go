package parameterization

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ktesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wffake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/server/auth"
)

func TestTemplateClosureFreezesNestedScopesAndRejectsStalePreview(t *testing.T) {
	ctx, store, service, revision := previewFixture(t)
	client := auth.GetWfClient(ctx).ArgoprojV1alpha1()
	root, err := client.WorkflowTemplates("lab").Get(ctx, "native", metav1.GetOptions{})
	require.NoError(t, err)
	root.Spec.Templates = []wfv1.Template{{Name: "main", DAG: &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{
		{Name: "first", TemplateRef: &wfv1.TemplateRef{Name: "library", Template: "sequence"}},
		{Name: "second", TemplateRef: &wfv1.TemplateRef{Name: "library", Template: "sequence"}, Depends: "first.Succeeded"},
	}}}}
	_, err = client.WorkflowTemplates("lab").Update(ctx, root, metav1.UpdateOptions{})
	require.NoError(t, err)
	library := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "library", Namespace: "lab", UID: "library-uid", ResourceVersion: "7"}, Spec: wfv1.WorkflowSpec{
		PodMetadata: &wfv1.Metadata{Labels: map[string]string{"library": "original"}},
		Templates: []wfv1.Template{
			{Name: "sequence", Steps: []wfv1.ParallelSteps{{Steps: []wfv1.WorkflowStep{{Name: "invoke", Template: "worker", OnExit: "finish", WithParam: "[1,2]"}}}}},
			{Name: "worker", Container: &corev1.Container{Image: "alpine:3.22", Command: []string{"echo", "original"}}},
			{Name: "finish", Steps: []wfv1.ParallelSteps{{Steps: []wfv1.WorkflowStep{{Name: "audit", TemplateRef: &wfv1.TemplateRef{Name: "common", Template: "record", ClusterScope: true}}}}}},
		},
	}}
	_, err = client.WorkflowTemplates("lab").Create(ctx, library, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = client.ClusterWorkflowTemplates().Create(ctx, &wfv1.ClusterWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "common", UID: "common-uid", ResourceVersion: "2"}, Spec: wfv1.WorkflowSpec{Templates: []wfv1.Template{{Name: "record", Container: &corev1.Container{Image: "alpine:3.22", Command: []string{"echo", "audit"}}}}}}, metav1.CreateOptions{})
	require.NoError(t, err)
	preview, err := store.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.Len(t, preview.Workflow.Spec.Templates, 5)
	main := preview.Workflow.Spec.Templates[0]
	require.Equal(t, main.DAG.Tasks[0].Template, main.DAG.Tasks[1].Template)
	require.Nil(t, main.DAG.Tasks[0].TemplateRef)
	require.Equal(t, "first.Succeeded", main.DAG.Tasks[1].Depends)
	sequence := preview.Workflow.GetTemplateByName(main.DAG.Tasks[0].Template)
	invoke := sequence.Steps[0].Steps[0]
	require.Equal(t, "[1,2]", invoke.WithParam)
	worker := preview.Workflow.GetTemplateByName(invoke.Template)
	require.Equal(t, "original", worker.Metadata.Labels["library"])
	require.Equal(t, []string{"echo", "original"}, worker.Container.Command)
	//nolint:staticcheck // Exercise freezing of existing workflows that still use onExit.
	finish := preview.Workflow.GetTemplateByName(invoke.OnExit)
	require.NotNil(t, preview.Workflow.GetTemplateByName(finish.Steps[0].Steps[0].Template))
	require.Contains(t, preview.DefinitionVersions, "templateRef/ClusterWorkflowTemplate/common")
	again, err := store.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.Equal(t, preview.ExecutionDigest, again.ExecutionDigest)
	// An implementation change, even if a broken/fake API fails to advance RV,
	// invalidates preview. The first snapshot remains independent of the source.
	library.Spec.Templates[1].Container.Command = []string{"echo", "changed"}
	_, err = client.WorkflowTemplates("lab").Update(ctx, library, metav1.UpdateOptions{})
	require.NoError(t, err)
	changed, err := store.Preview(ctx, "standard", revision)
	require.NoError(t, err)
	require.NotEqual(t, preview.ExecutionDigest, changed.ExecutionDigest)
	require.Equal(t, []string{"echo", "original"}, worker.Container.Command)
	_, err = (Handler{Workflows: service}).submit(ctx, store, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "stale-closure"})
	require.ErrorContains(t, err, "stale")
	require.Zero(t, service.createCalls)
}

func TestTemplateClosurePermissionsMissingAndDynamicReferences(t *testing.T) {
	for _, tc := range []struct {
		name    string
		ref     *wfv1.TemplateRef
		deny    bool
		message string
	}{
		{"dynamic", &wfv1.TemplateRef{Name: "{{workflow.parameters.library}}", Template: "main"}, false, "static"},
		{"missing", &wfv1.TemplateRef{Name: "missing", Template: "main"}, false, "not found"},
		{"denied", &wfv1.TemplateRef{Name: "restricted", Template: "main", ClusterScope: true}, true, "forbidden"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _, _, _ := previewFixture(t)
			client := auth.GetWfClient(ctx).(*wffake.Clientset)
			if tc.deny {
				client.PrependReactor("get", "clusterworkflowtemplates", func(a ktesting.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "argoproj.io", Resource: "clusterworkflowtemplates"}, "restricted", fmt.Errorf("denied"))
				})
			}
			wf := &wfv1.Workflow{Spec: wfv1.WorkflowSpec{Templates: []wfv1.Template{{Name: "main", DAG: &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{{Name: "use", TemplateRef: tc.ref}}}}}}}
			err := freezeTemplateClosure(ctx, "lab", wf, map[string]string{})
			require.ErrorContains(t, err, tc.message)
		})
	}
}

func TestTemplateClosurePreservesRecursionInlineAndHooks(t *testing.T) {
	ctx, _, _, _ := previewFixture(t)
	lib := &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "recursive", Namespace: "lab"}, Spec: wfv1.WorkflowSpec{Templates: []wfv1.Template{
		{Name: "loop", DAG: &wfv1.DAGTemplate{Tasks: []wfv1.DAGTask{{Name: "next", Template: "loop", When: "false"}, {Name: "inline", Inline: &wfv1.Template{Steps: []wfv1.ParallelSteps{{Steps: []wfv1.WorkflowStep{{Name: "end", Template: "done"}}}}}}}}},
		{Name: "done", Container: &corev1.Container{Image: "alpine:3.22"}},
	}}}
	_, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().WorkflowTemplates("lab").Create(ctx, lib, metav1.CreateOptions{})
	require.NoError(t, err)
	wf := &wfv1.Workflow{Spec: wfv1.WorkflowSpec{Templates: []wfv1.Template{{Name: "main", Container: &corev1.Container{Image: "alpine:3.22"}}}, Hooks: wfv1.LifecycleHooks{wfv1.ExitLifecycleEvent: {TemplateRef: &wfv1.TemplateRef{Name: "recursive", Template: "loop"}}}}}
	require.NoError(t, freezeTemplateClosure(ctx, "lab", wf, map[string]string{}))
	require.Len(t, wf.Spec.Templates, 3)
	loop := wf.GetTemplateByName(wf.Spec.Hooks[wfv1.ExitLifecycleEvent].Template)
	require.Equal(t, loop.Name, loop.DAG.Tasks[0].Template)
	require.Equal(t, "false", loop.DAG.Tasks[0].When)
	require.NotNil(t, wf.GetTemplateByName(loop.DAG.Tasks[1].Inline.Steps[0].Steps[0].Template))
}
