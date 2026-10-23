package parameterization

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clienttesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
)

func TestActionReferenceFreezesCallerScopedImplementation(t *testing.T) {
	for _, cluster := range []bool{false, true} {
		t.Run(fmt.Sprint(cluster), func(t *testing.T) {
			ctx, s, w, revision := previewFixture(t)
			wc := auth.GetWfClient(ctx)
			library := wfv1.WorkflowSpec{PodMetadata: &wfv1.Metadata{Labels: map[string]string{"origin": "library"}}, Templates: []wfv1.Template{{Name: "emit", Container: &corev1.Container{Image: "alpine:3.22", Command: []string{"echo", "frozen"}}}}}
			if cluster {
				_, err := wc.ArgoprojV1alpha1().ClusterWorkflowTemplates().Create(ctx, &wfv1.ClusterWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "library", UID: "source", ResourceVersion: "1"}, Spec: library}, metav1.CreateOptions{})
				require.NoError(t, err)
			} else {
				_, err := wc.ArgoprojV1alpha1().WorkflowTemplates("lab").Create(ctx, &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: "library", Namespace: "lab", UID: "source", ResourceVersion: "1"}, Spec: library}, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			action := ActionDefinition{InputSchema: map[string]any{"type": "object"}, OutputSchema: map[string]any{"type": "object"}, ImplementationRef: &wfv1.TemplateRef{Name: "library", Template: "emit", ClusterScope: cluster}}
			_, err := s.resource("actiondefinitions").Create(ctx, object("ActionDefinition", "ref", "lab", action), metav1.CreateOptions{})
			require.NoError(t, err)
			_, err = s.resource("workflowactionbindings").Create(ctx, object("WorkflowActionBinding", "calls", "lab", ActionBinding{WorkflowTemplate: "native", Calls: []ActionCall{{Template: "main", ActionDefinition: "ref"}}}), metav1.CreateOptions{})
			require.NoError(t, err)
			binding, err := s.get(ctx, "workflowparameterbindings", "standard")
			require.NoError(t, err)
			require.NoError(t, unstructured.SetNestedField(binding.Object, "calls", "spec", "actionBinding"))
			_, err = s.resource("workflowparameterbindings").Update(ctx, binding, metav1.UpdateOptions{})
			require.NoError(t, err)
			preview, err := s.Preview(ctx, "standard", revision)
			require.NoError(t, err)
			require.Nil(t, preview.Definitions.Actions["ref"].Template, "definition preserves the source reference")
			require.Equal(t, "library", preview.Definitions.Actions["ref"].ImplementationRef.Name)
			require.Equal(t, []string{"echo", "frozen"}, preview.Workflow.Spec.Templates[0].Container.Command)
			require.Equal(t, "library", preview.Workflow.Spec.Templates[0].Metadata.Labels["origin"])
			key := "templateRef/WorkflowTemplate/lab/library"
			resource := "workflowtemplates"
			if cluster {
				key = "templateRef/ClusterWorkflowTemplate/library"
				resource = "clusterworkflowtemplates"
			}
			require.Contains(t, preview.DefinitionVersions, key)
			closure := &templateClosure{namespace: "lab", sources: map[string]wfv1.TemplateHolder{}, versions: map[string]string{key: "different-source"}}
			_, err = closure.source(ctx, action.ImplementationRef)
			require.ErrorContains(t, err, "changed while resolving", "actions and native templateRefs cannot freeze different versions of the same library")
			if cluster {
				x, sourceErr := wc.ArgoprojV1alpha1().ClusterWorkflowTemplates().Get(ctx, "library", metav1.GetOptions{})
				require.NoError(t, sourceErr)
				require.Nil(t, x.Spec.Templates[0].Metadata.Labels, "preview must not mutate the library")
				x.Spec.Templates[0].Container.Command[1] = "changed"
				_, sourceErr = wc.ArgoprojV1alpha1().ClusterWorkflowTemplates().Update(ctx, x, metav1.UpdateOptions{})
				require.NoError(t, sourceErr)
			} else {
				x, sourceErr := wc.ArgoprojV1alpha1().WorkflowTemplates("lab").Get(ctx, "library", metav1.GetOptions{})
				require.NoError(t, sourceErr)
				require.Nil(t, x.Spec.Templates[0].Metadata.Labels)
				x.Spec.Templates[0].Container.Command[1] = "changed"
				_, sourceErr = wc.ArgoprojV1alpha1().WorkflowTemplates("lab").Update(ctx, x, metav1.UpdateOptions{})
				require.NoError(t, sourceErr)
			}
			_, err = (&Handler{Workflows: w}).submit(ctx, s, SubmitRequest{Binding: "standard", Revision: revision, ExecutionDigest: preview.ExecutionDigest, IdempotencyKey: "stale"})
			require.ErrorContains(t, err, "stale")
			require.Zero(t, w.createCalls)
			wc.(interface {
				PrependReactor(string, string, clienttesting.ReactionFunc)
			}).PrependReactor("get", resource, func(a clienttesting.Action) (bool, runtime.Object, error) {
				if a.(clienttesting.GetAction).GetName() != "library" {
					return false, nil, nil
				}
				return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: "argoproj.io", Resource: resource}, "library", fmt.Errorf("caller denied"))
			})
			_, err = s.Preview(ctx, "standard", revision)
			require.True(t, apierrors.IsForbidden(err), "holding a binding must not confer library access")
		})
	}
}

func TestActionReferenceContractAndDefinitionStatus(t *testing.T) {
	schema := map[string]any{"type": "object"}
	ref := &wfv1.TemplateRef{Name: "library", Template: "emit"}
	leaf := &wfv1.Template{Name: "emit", Container: &corev1.Container{Image: "alpine"}}
	for _, item := range []struct {
		name     string
		template *wfv1.Template
		ref      *wfv1.TemplateRef
		message  string
	}{
		{"none", nil, nil, "exactly one"}, {"both", leaf, ref, "exactly one"},
		{"dynamic", nil, &wfv1.TemplateRef{Name: "{{workflow.parameters.library}}", Template: "emit"}, "static"},
		{"composite", &wfv1.Template{Name: "dag", DAG: &wfv1.DAGTemplate{}}, nil, "leaf"},
	} {
		t.Run(item.name, func(t *testing.T) {
			_, err := resolveAction(ActionDefinition{InputSchema: schema, OutputSchema: schema, Template: item.template, ImplementationRef: item.ref}, func(*wfv1.TemplateRef) (*wfv1.Template, error) {
				t.Fatal("invalid target must fail before lookup")
				return nil, nil
			})
			require.ErrorContains(t, err, item.message)
		})
	}
	action := object("ActionDefinition", "action", "lab", ActionDefinition{InputSchema: schema, OutputSchema: schema, ImplementationRef: ref})
	library := object("WorkflowTemplate", "library", "lab", wfv1.WorkflowSpec{Templates: []wfv1.Template{*leaf}})
	library.SetUID("one")
	library.SetResourceVersion("1")
	lookup := func(key DefinitionKey) (*unstructured.Unstructured, error) {
		require.Equal(t, DefinitionKey{"workflowtemplates", "library"}, key)
		return library, nil
	}
	status := EvaluateDefinition(action, lookup)
	require.Empty(t, status.Errors)
	require.Len(t, status.ResolvedReferences, 1)
	require.NoError(t, unstructured.SetNestedField(action.Object, "missing", "spec", "implementationRef", "template"))
	require.Contains(t, EvaluateDefinition(action, lookup).Errors[0].Message, "absent")
	require.NoError(t, unstructured.SetNestedField(action.Object, "emit", "spec", "implementationRef", "template"))
	require.NoError(t, unstructured.SetNestedMap(action.Object, map[string]any{"required": map[string]any{"type": "string"}}, "spec", "inputSchema", "properties"))
	require.Contains(t, EvaluateDefinition(action, lookup).Errors[0].Message, "contract parameter")
}
