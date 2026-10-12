package parameterization

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"

	p "github.com/argoproj/argo-workflows/v4/util/parameterization"
)

// This client-go fake predates CreateOptions recording; honor dry-run at the
// transport boundary while leaving all persisted objects in its real tracker.
type publicationDryRunClient struct{ dynamic.Interface }
type publicationNamespace struct {
	dynamic.NamespaceableResourceInterface
}
type publicationResource struct{ dynamic.ResourceInterface }

func (c publicationDryRunClient) Resource(gvr schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return publicationNamespace{c.Interface.Resource(gvr)}
}
func (r publicationNamespace) Namespace(ns string) dynamic.ResourceInterface {
	return publicationResource{r.NamespaceableResourceInterface.Namespace(ns)}
}
func (r publicationResource) Create(ctx context.Context, obj *unstructured.Unstructured, opts metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	if len(opts.DryRun) > 0 {
		return obj.DeepCopy(), nil
	}
	clone := obj.DeepCopy()
	clone.SetUID(types.UID("published-" + obj.GetName()))
	clone.SetResourceVersion("8")
	return r.ResourceInterface.Create(ctx, clone, opts, subresources...)
}

func publicationFixture() (Store, *dynamicfake.FakeDynamicClient, *unstructured.Unstructured) {
	source := object("ParameterTemplate", "draft", "lab", map[string]any{"dialect": p.Dialect, "schema": map[string]any{
		"type": "object", "properties": map[string]any{"count": map[string]any{"type": "integer", "default": int64(1)}}}})
	source.SetUID("source-one")
	source.SetResourceVersion("7")
	source.SetGeneration(1)
	c := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		{Group: p.Group, Version: p.Version, Resource: "parametertemplates"}: "ParameterTemplateList",
	}, source)
	return Store{Client: publicationDryRunClient{c}, Namespace: "lab", Actor: "author"}, c, source
}

func publicationRequest(source *unstructured.Unstructured, name string) DefinitionPublishRequest {
	return DefinitionPublishRequest{Source: source.GetName(), UID: string(source.GetUID()), SourceSpec: source.Object["spec"].(map[string]any), Name: name, Message: "reviewed version"}
}

func TestPublicationImmutableNamedVersionAndLostResponse(t *testing.T) {
	s, c, source := publicationFixture()
	ctx := context.Background()
	req := publicationRequest(source, "model-v1")
	req.ValidateOnly = true
	_, err := s.publishDefinition(ctx, "parametertemplates", req)
	require.NoError(t, err)
	_, err = s.get(ctx, "parametertemplates", req.Name)
	require.True(t, apierrors.IsNotFound(err))
	req.ValidateOnly = false
	version, err := s.publishDefinition(ctx, "parametertemplates", req)
	require.NoError(t, err)
	require.Equal(t, source.Object["spec"], version.Object["spec"])
	require.True(t, isPublishedDefinition(version))
	require.Equal(t, "author", version.GetAnnotations()[definitionPublicationPrefix+"publisher"])
	original, err := s.get(ctx, "parametertemplates", "draft")
	require.NoError(t, err)
	require.Equal(t, source.Object, original.Object)
	_, err = s.writeDefinition(ctx, "parametertemplates", req.Name, DefinitionWrite{UID: string(version.GetUID()), ResourceVersion: version.GetResourceVersion(), Spec: map[string]any{"new": true}})
	require.ErrorContains(t, err, "immutable")
	require.NoError(t, s.resource("parametertemplates").Delete(ctx, "draft", metav1.DeleteOptions{}))
	before := len(c.Actions())
	recovered, err := s.publishDefinition(ctx, "parametertemplates", req)
	require.NoError(t, err)
	require.Equal(t, version.GetUID(), recovered.GetUID())
	for _, action := range c.Actions()[before:] {
		require.Equal(t, "get", action.GetVerb())
	}
	versions, err := s.definitionVersions(ctx, "parametertemplates", "source-one", "")
	require.NoError(t, err)
	require.Len(t, versions.Items, 1)
	require.Equal(t, "PartialObjectMetadataList", versions.GetKind())
	require.NotContains(t, versions.Items[0].Object, "spec")
}

func TestPublicationRejectsConflictsInvalidSchemasAndForbiddenSource(t *testing.T) {
	s, c, source := publicationFixture()
	ctx := context.Background()
	req := publicationRequest(source, "model-v1")
	wrong := req
	wrong.UID = "replaced"
	_, err := s.publishDefinition(ctx, "parametertemplates", wrong)
	require.True(t, apierrors.IsConflict(err))
	current := source.DeepCopy()
	current.SetResourceVersion("999") // status-only update must not reject the reviewed spec
	_, err = s.resource("parametertemplates").UpdateStatus(ctx, current, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.publishDefinition(ctx, "parametertemplates", req)
	require.NoError(t, err)
	wrong = req
	wrong.Message = "another intent"
	_, err = s.publishDefinition(ctx, "parametertemplates", wrong)
	require.True(t, apierrors.IsConflict(err))
	require.NoError(t, unstructured.SetNestedField(current.Object, "unsupported", "spec", "schema", "type"))
	_, err = s.resource("parametertemplates").Update(ctx, current, metav1.UpdateOptions{})
	require.NoError(t, err)
	wrong = publicationRequest(current, "invalid")
	_, err = s.publishDefinition(ctx, "parametertemplates", wrong)
	require.ErrorContains(t, err, "cannot be published")
	_, err = s.get(ctx, "parametertemplates", "invalid")
	require.True(t, apierrors.IsNotFound(err))
	_, err = s.publishDefinition(ctx, "workflowsubmissions", req)
	require.Error(t, err)
	c.PrependReactor("get", "parametertemplates", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: p.Group, Resource: "parametertemplates"}, action.(clienttesting.GetAction).GetName(), nil)
	})
	_, err = s.publishDefinition(ctx, "parametertemplates", req)
	require.True(t, apierrors.IsForbidden(err))
}

func TestDefinitionVersionDiffStablePaginationAndChangedDraft(t *testing.T) {
	s, _, source := publicationFixture()
	ctx := context.Background()
	_, err := s.publishDefinition(ctx, "parametertemplates", publicationRequest(source, "model-v1"))
	require.NoError(t, err)
	current := source.DeepCopy()
	require.NoError(t, unstructured.SetNestedField(current.Object, int64(3), "spec", "schema", "properties", "count", "default"))
	require.NoError(t, unstructured.SetNestedField(current.Object, "Version two", "spec", "schema", "title"))
	_, err = s.resource("parametertemplates").Update(ctx, current, metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = s.publishDefinition(ctx, "parametertemplates", publicationRequest(current, "model-v2"))
	require.NoError(t, err)
	req := DiffRequest{Left: "model-v1", Right: "model-v2", Limit: 1}
	page, err := s.definitionDiff(ctx, "parametertemplates", req)
	require.NoError(t, err)
	require.Len(t, page["changes"], 1)
	req.Continue = page["pagination"].(map[string]any)["continue"].(string)
	require.NotEmpty(t, req.Continue)
	page, err = s.definitionDiff(ctx, "parametertemplates", req)
	require.NoError(t, err)
	require.Len(t, page["changes"], 1)
	req.Right = "draft"
	_, err = s.definitionDiff(ctx, "parametertemplates", req)
	require.ErrorContains(t, err, "restart comparison")
}
