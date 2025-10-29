package templateresolution

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	fakewfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func createWorkflowTemplate(ctx context.Context, wfClientset wfclientset.Interface, yamlStr string) error {
	wftmpl := unmarshalWftmpl(yamlStr)
	_, err := wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault).Create(ctx, wftmpl, metav1.CreateOptions{})
	if err != nil && apierr.IsAlreadyExists(err) {
		return nil
	}
	return err
}

// Deprecated
func unmarshalWftmpl(yamlStr string) *wfv1.WorkflowTemplate {
	return wfv1.MustUnmarshalWorkflowTemplate(yamlStr)
}

var someWorkflowTemplateYaml = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: some-workflow-template
spec:
  templates:
  - name: local-whalesay
    steps:
      - - name: step
          template: whalesay
  - name: another-whalesay
    steps:
      - - name: step
          templateRef:
            name: another-workflow-template
            template: whalesay
  - name: whalesay
    container:
      image: docker/whalesay
  - name: nested-whalesay-with-arguments
    steps:
      - - name: step
  - name: whalesay-with-arguments
    steps:
      - - name: step
          template: whalesay-with-arguments-inner
  - name: whalesay-with-arguments-inner
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
  - name: infinite-local-loop-whalesay
    steps:
      - - name: step
          template: infinite-local-loop-whalesay
  - name: infinite-loop-whalesay
    steps:
      - - name: step
          templateRef:
            name: some-workflow-template
            template: infinite-loop-whalesay
`

var anotherWorkflowTemplateYaml = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: another-workflow-template
spec:
  templates:
  - name: whalesay
    container:
      image: docker/whalesay
`

var baseWorkflowTemplateYaml = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: base-workflow-template
spec:
  templates:
  - name: whalesay
    container:
      image: docker/whalesay
`

func TestGetTemplateByName(t *testing.T) {
	wfClientset := fakewfclientset.NewSimpleClientset()
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	ctx := logging.TestContext(t.Context())
	log := logging.RequireLoggerFromContext(ctx)
	tplCtx := NewContextFromClientSet(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault), wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates(), wftmpl, nil, log)

	tmpl, err := tplCtx.GetTemplateByName(ctx, "whalesay")
	require.NoError(t, err)
	assert.Equal(t, "whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	_, err = tplCtx.GetTemplateByName(ctx, "unknown")
	require.EqualError(t, err, "template unknown not found")
}

func TestGetTemplateFromRef(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wfClientset := fakewfclientset.NewSimpleClientset()
	err := createWorkflowTemplate(ctx, wfClientset, anotherWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	err = createWorkflowTemplate(ctx, wfClientset, someWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	log := logging.RequireLoggerFromContext(ctx)
	tplCtx := NewContextFromClientSet(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault), wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates(), wftmpl, nil, log)

	// Get the template of existing template reference.
	tmplRef := wfv1.TemplateRef{Name: "some-workflow-template", Template: "whalesay"}
	tmpl, err := tplCtx.GetTemplateFromRef(ctx, &tmplRef)
	require.NoError(t, err)
	assert.Equal(t, "whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	// Get the template of unexisting template reference.
	tmplRef = wfv1.TemplateRef{Name: "unknown-workflow-template", Template: "whalesay"}
	_, err = tplCtx.GetTemplateFromRef(ctx, &tmplRef)
	require.EqualError(t, err, "workflow template unknown-workflow-template not found")

	// Get the template of unexisting template name of existing template reference.
	tmplRef = wfv1.TemplateRef{Name: "some-workflow-template", Template: "unknown"}
	_, err = tplCtx.GetTemplateFromRef(ctx, &tmplRef)
	require.EqualError(t, err, "template unknown not found in workflow template some-workflow-template")
}

func TestGetTemplate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wfClientset := fakewfclientset.NewSimpleClientset()
	err := createWorkflowTemplate(ctx, wfClientset, anotherWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	err = createWorkflowTemplate(ctx, wfClientset, someWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	log := logging.RequireLoggerFromContext(ctx)
	tplCtx := NewContextFromClientSet(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault), wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates(), wftmpl, nil, log)

	// Get the template of existing template name.
	tmplHolder := wfv1.WorkflowStep{Template: "whalesay"}
	tmpl, err := tplCtx.GetTemplate(ctx, &tmplHolder)
	require.NoError(t, err)
	assert.Equal(t, "whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	// Get the template of unexisting template name.
	tmplHolder = wfv1.WorkflowStep{Template: "unexisting"}
	_, err = tplCtx.GetTemplate(ctx, &tmplHolder)
	require.EqualError(t, err, "template unexisting not found")

	// Get the template of existing template reference.
	tmplHolder = wfv1.WorkflowStep{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "whalesay"}}
	tmpl, err = tplCtx.GetTemplate(ctx, &tmplHolder)
	require.NoError(t, err)
	assert.Equal(t, "whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	// Get the template of unexisting template reference.
	tmplHolder = wfv1.WorkflowStep{TemplateRef: &wfv1.TemplateRef{Name: "unknown-workflow-template", Template: "whalesay"}}
	_, err = tplCtx.GetTemplate(ctx, &tmplHolder)
	require.EqualError(t, err, "workflow template unknown-workflow-template not found")
}

func TestGetCurrentTemplateBase(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wfClientset := fakewfclientset.NewSimpleClientset()
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	log := logging.RequireLoggerFromContext(ctx)
	tplCtx := NewContextFromClientSet(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault), wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates(), wftmpl, nil, log)

	// Get the template base of existing template name.
	tmplBase := tplCtx.GetCurrentTemplateBase()
	wftmpl, ok := tmplBase.(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "base-workflow-template", wftmpl.Name)
}

func TestWithTemplateHolder(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wfClientset := fakewfclientset.NewSimpleClientset()
	err := createWorkflowTemplate(ctx, wfClientset, anotherWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	err = createWorkflowTemplate(ctx, wfClientset, someWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	log := logging.RequireLoggerFromContext(ctx)
	tplCtx := NewContextFromClientSet(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault), wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates(), wftmpl, nil, log)

	var tmplGetter wfv1.TemplateHolder
	// Get the template base of existing template name.
	tmplHolder := wfv1.WorkflowStep{Template: "whalesay"}
	newCtx, err := tplCtx.WithTemplateHolder(ctx, &tmplHolder)
	require.NoError(t, err)
	tmplGetter, ok := newCtx.GetCurrentTemplateBase().(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "base-workflow-template", tmplGetter.GetName())

	// Get the template base of unexisting template name.
	tmplHolder = wfv1.WorkflowStep{Template: "unknown"}
	newCtx, err = tplCtx.WithTemplateHolder(ctx, &tmplHolder)
	require.NoError(t, err)
	tmplGetter, ok = newCtx.GetCurrentTemplateBase().(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "base-workflow-template", tmplGetter.GetName())

	// Get the template base of existing template reference.
	tmplHolder = wfv1.WorkflowStep{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "whalesay"}}
	newCtx, err = tplCtx.WithTemplateHolder(ctx, &tmplHolder)
	require.NoError(t, err)
	tmplGetter, ok = newCtx.GetCurrentTemplateBase().(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "some-workflow-template", tmplGetter.GetName())

	// Get the template base of unexisting template reference.
	tmplHolder = wfv1.WorkflowStep{TemplateRef: &wfv1.TemplateRef{Name: "unknown-workflow-template", Template: "whalesay"}}
	_, err = tplCtx.WithTemplateHolder(ctx, &tmplHolder)
	require.EqualError(t, err, "workflowtemplates.argoproj.io \"unknown-workflow-template\" not found")
}

func TestResolveTemplate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wfClientset := fakewfclientset.NewSimpleClientset()
	err := createWorkflowTemplate(ctx, wfClientset, anotherWorkflowTemplateYaml)
	require.NoError(t, err)

	err = createWorkflowTemplate(ctx, wfClientset, someWorkflowTemplateYaml)
	require.NoError(t, err)

	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	log := logging.RequireLoggerFromContext(ctx)
	tplCtx := NewContextFromClientSet(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault), wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates(), wftmpl, nil, log)

	// Get the template of template name.
	tmplHolder := wfv1.WorkflowStep{Template: "whalesay"}
	tplCtx, tmpl, _, err := tplCtx.ResolveTemplate(ctx, &tmplHolder)
	require.NoError(t, err)
	wftmpl, ok := tplCtx.tmplBase.(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "base-workflow-template", wftmpl.Name)
	assert.Equal(t, "whalesay", tmpl.Name)

	var tmplGetter wfv1.TemplateHolder
	// Get the template of template reference.
	tmplHolder = wfv1.WorkflowStep{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "whalesay"}}
	tplCtx, tmpl, _, err = tplCtx.ResolveTemplate(ctx, &tmplHolder)
	require.NoError(t, err)

	tmplGetter, ok = tplCtx.tmplBase.(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "some-workflow-template", tmplGetter.GetName())
	assert.Equal(t, "whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	// Get the template of local nested template reference.
	tmplHolder = wfv1.WorkflowStep{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "local-whalesay"}}
	tplCtx, tmpl, _, err = tplCtx.ResolveTemplate(ctx, &tmplHolder)
	require.NoError(t, err)

	tmplGetter, ok = tplCtx.tmplBase.(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "some-workflow-template", tmplGetter.GetName())
	assert.Equal(t, "local-whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Steps)

	// Get the template of nested template reference.
	tmplHolder = wfv1.WorkflowStep{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "another-whalesay"}}
	tplCtx, tmpl, _, err = tplCtx.ResolveTemplate(ctx, &tmplHolder)
	require.NoError(t, err)

	tmplGetter, ok = tplCtx.tmplBase.(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "some-workflow-template", tmplGetter.GetName())
	assert.Equal(t, "another-whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Steps)

	// Get the template of template reference with arguments.
	tmplHolder = wfv1.WorkflowStep{
		TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "whalesay-with-arguments"},
	}
	tplCtx, tmpl, _, err = tplCtx.ResolveTemplate(ctx, &tmplHolder)
	require.NoError(t, err)

	tmplGetter, ok = tplCtx.tmplBase.(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "some-workflow-template", tmplGetter.GetName())
	assert.Equal(t, "whalesay-with-arguments", tmpl.Name)

	// Get the template of nested template reference with arguments.
	tmplHolder = wfv1.WorkflowStep{
		TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "nested-whalesay-with-arguments"},
	}
	tplCtx, tmpl, _, err = tplCtx.ResolveTemplate(ctx, &tmplHolder)
	require.NoError(t, err)

	tmplGetter, ok = tplCtx.tmplBase.(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "some-workflow-template", tmplGetter.GetName())
	assert.Equal(t, "nested-whalesay-with-arguments", tmpl.Name)
}

func TestWithTemplateBase(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wfClientset := fakewfclientset.NewSimpleClientset()
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	log := logging.RequireLoggerFromContext(ctx)
	tplCtx := NewContextFromClientSet(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault), wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates(), wftmpl, nil, log)

	anotherWftmpl := unmarshalWftmpl(anotherWorkflowTemplateYaml)

	// Get the template base of existing template name.
	newCtx := tplCtx.WithTemplateBase(anotherWftmpl)
	wftmpl, ok := newCtx.tmplBase.(*wfv1.WorkflowTemplate)
	require.True(t, ok, "tmplBase is not a WorkflowTemplate")
	assert.Equal(t, "another-workflow-template", wftmpl.Name)
}

func TestOnWorkflowTemplate(t *testing.T) {
	wfClientset := fakewfclientset.NewSimpleClientset()
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	ctx := logging.TestContext(t.Context())
	log := logging.RequireLoggerFromContext(ctx)
	tplCtx := NewContextFromClientSet(wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault), wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates(), wftmpl, nil, log)

	err := createWorkflowTemplate(ctx, wfClientset, anotherWorkflowTemplateYaml)
	require.NoError(t, err)

	// Get the template base of existing template name.
	newCtx, err := tplCtx.WithWorkflowTemplate(ctx, "another-workflow-template")
	require.NoError(t, err)
	tmpl := newCtx.tmplBase.GetTemplateByName("whalesay")
	assert.NotNil(t, tmpl)
}

// TestGetTemplateFromRefWithPodMetadataAndMissingTemplate tests the bug where
// GetTemplateFromRef causes a nil pointer dereference when:
// 1. A WorkflowTemplate has podMetadata defined
// 2. A templateRef references a template name that doesn't exist in that WorkflowTemplate
func TestGetTemplateFromRefWithPodMetadataAndMissingTemplate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wfClientset := fakewfclientset.NewSimpleClientset()

	// Create a WorkflowTemplate with podMetadata but without the template "nonexistent"
	workflowTemplateWithPodMetadata := `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: template-with-podmetadata
spec:
  podMetadata:
    labels:
      example-label: example-value
    annotations:
      example-annotation: example-value
  templates:
  - name: existing-template
    container:
      image: alpine:latest
      command: [echo, hello]
`

	err := createWorkflowTemplate(ctx, wfClientset, workflowTemplateWithPodMetadata)
	require.NoError(t, err)

	// Create a base workflow template to use as context
	baseWftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	log := logging.RequireLoggerFromContext(ctx)
	tplCtx := NewContextFromClientSet(
		wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault),
		wfClientset.ArgoprojV1alpha1().ClusterWorkflowTemplates(),
		baseWftmpl,
		nil,
		log,
	)

	// Try to get a template that doesn't exist from a WorkflowTemplate that HAS podMetadata
	tmplRef := wfv1.TemplateRef{
		Name:     "template-with-podmetadata",
		Template: "nonexistent-template",
	}

	_, err = tplCtx.GetTemplateFromRef(ctx, &tmplRef)

	require.Error(t, err)
	require.Contains(t, err.Error(), "template nonexistent-template not found in workflow template template-with-podmetadata")
}
