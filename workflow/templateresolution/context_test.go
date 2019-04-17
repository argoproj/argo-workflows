package templateresolution

import (
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	fakewfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned/fake"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createWorkflowTemplate(wfClientset wfclientset.Interface, yamlStr string) error {
	wftmpl := unmarshalWftmpl(yamlStr)
	_, err := wfClientset.ArgoprojV1alpha1().WorkflowTemplates(metav1.NamespaceDefault).Create(wftmpl)
	if err != nil && apierr.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func unmarshalWftmpl(yamlStr string) *wfv1.WorkflowTemplate {
	var wftmpl wfv1.WorkflowTemplate
	err := yaml.Unmarshal([]byte(yamlStr), &wftmpl)
	if err != nil {
		panic(err)
	}
	return &wftmpl
}

var someWorkflowTemplateYaml = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: some-workflow-template
spec:
  templates:
  - name: local-whalesay
    template: whalesay
  - name: another-whalesay
    templateRef:
      name: another-workflow-template
      template: whalesay
  - name: whalesay
    container:
      image: docker/whalesay
  - name: nested-whalesay-with-arguments
    template: whalesay-with-arguments
    inputs:
      parameters:
      - name: message
    arguments:
      parameters:
      - name: message
        value: "{{inputs.parameters.message}}-bar"
  - name: whalesay-with-arguments
    template: whalesay-with-arguments-inner
    inputs:
      parameters:
      - name: message
    arguments:
      parameters:
      - name: message
        value: "{{inputs.parameters.message}}-foo"
  - name: whalesay-with-arguments-inner
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
  - name: infinite-local-loop-whalesay
    template: infinite-local-loop-whalesay
  - name: infinite-loop-whalesay
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
	ctx := NewContext(metav1.NamespaceDefault, wfClientset, wftmpl)

	tmpl, err := ctx.GetTemplateByName("whalesay")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	assert.Equal(t, "whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	_, err = ctx.GetTemplateByName("unknown")
	assert.EqualError(t, err, "template unknown not found")
}

func TestGetTemplateFromRef(t *testing.T) {
	wfClientset := fakewfclientset.NewSimpleClientset()
	err := createWorkflowTemplate(wfClientset, anotherWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	err = createWorkflowTemplate(wfClientset, someWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	ctx := NewContext(metav1.NamespaceDefault, wfClientset, wftmpl)

	// Get the template of existing template reference.
	tmplRef := wfv1.TemplateRef{Name: "some-workflow-template", Template: "whalesay"}
	tmpl, err := ctx.GetTemplateFromRef(&tmplRef)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	assert.Equal(t, "whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	// Get the template of unexisting template reference.
	tmplRef = wfv1.TemplateRef{Name: "unknown-workflow-template", Template: "whalesay"}
	_, err = ctx.GetTemplateFromRef(&tmplRef)
	assert.EqualError(t, err, "workflow template unknown-workflow-template not found")

	// Get the template of unexisting template name of existing template reference.
	tmplRef = wfv1.TemplateRef{Name: "some-workflow-template", Template: "unknown"}
	_, err = ctx.GetTemplateFromRef(&tmplRef)
	assert.EqualError(t, err, "template unknown not found in workflow template some-workflow-template")
}

func TestGetTemplate(t *testing.T) {
	wfClientset := fakewfclientset.NewSimpleClientset()
	err := createWorkflowTemplate(wfClientset, anotherWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	err = createWorkflowTemplate(wfClientset, someWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	ctx := NewContext(metav1.NamespaceDefault, wfClientset, wftmpl)

	// Get the template of existing template name.
	tmplHolder := wfv1.Template{Template: "whalesay"}
	tmpl, err := ctx.GetTemplate(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	assert.Equal(t, "whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	// Get the template of unexisting template name.
	tmplHolder = wfv1.Template{}
	_, err = ctx.GetTemplate(&tmplHolder)
	assert.EqualError(t, err, "template  not found")

	// Get the template of existing template reference.
	tmplHolder = wfv1.Template{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "whalesay"}}
	tmpl, err = ctx.GetTemplate(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	assert.Equal(t, "whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	// Get the template of unexisting template reference.
	tmplHolder = wfv1.Template{TemplateRef: &wfv1.TemplateRef{Name: "unknown-workflow-template", Template: "whalesay"}}
	_, err = ctx.GetTemplate(&tmplHolder)
	assert.EqualError(t, err, "workflow template unknown-workflow-template not found")
}

func TestGetTemplateBase(t *testing.T) {
	wfClientset := fakewfclientset.NewSimpleClientset()
	err := createWorkflowTemplate(wfClientset, anotherWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	err = createWorkflowTemplate(wfClientset, someWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	ctx := NewContext(metav1.NamespaceDefault, wfClientset, wftmpl)

	// Get the template base of existing template name.
	tmplHolder := wfv1.Template{Template: "whalesay"}
	tmplBase, err := ctx.GetTemplateBase(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	wftmpl, ok := tmplBase.(*wfv1.WorkflowTemplate)
	if !assert.True(t, ok) {
		t.Fatal(err)
	}
	assert.Equal(t, "base-workflow-template", wftmpl.Name)

	// Get the template base of unexisting template name.
	tmplHolder = wfv1.Template{Template: "unknown"}
	tmplBase, err = ctx.GetTemplateBase(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	wftmpl, ok = tmplBase.(*wfv1.WorkflowTemplate)
	if !assert.True(t, ok) {
		t.Fatal(err)
	}
	assert.Equal(t, "base-workflow-template", wftmpl.Name)

	// Get the template base of existing template reference.
	tmplHolder = wfv1.Template{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "whalesay"}}
	tmplBase, err = ctx.GetTemplateBase(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	wftmpl, ok = tmplBase.(*wfv1.WorkflowTemplate)
	if !assert.True(t, ok) {
		t.Fatal(err)
	}
	assert.Equal(t, "some-workflow-template", wftmpl.Name)

	// Get the template base of unexisting template reference.
	tmplHolder = wfv1.Template{TemplateRef: &wfv1.TemplateRef{Name: "unknown-workflow-template", Template: "whalesay"}}
	_, err = ctx.GetTemplateBase(&tmplHolder)
	assert.EqualError(t, err, "workflow template unknown-workflow-template not found")
}

func TestResolveTemplate(t *testing.T) {
	wfClientset := fakewfclientset.NewSimpleClientset()
	err := createWorkflowTemplate(wfClientset, anotherWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	err = createWorkflowTemplate(wfClientset, someWorkflowTemplateYaml)
	if err != nil {
		t.Fatal(err)
	}
	wftmpl := unmarshalWftmpl(baseWorkflowTemplateYaml)
	ctx := NewContext(metav1.NamespaceDefault, wfClientset, wftmpl)

	// Get the template of template name.
	tmplHolder := wfv1.Template{Template: "whalesay"}
	ctx, tmpl, err := ctx.ResolveTemplate(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	wftmpl, ok := ctx.tmplBase.(*wfv1.WorkflowTemplate)
	if !assert.True(t, ok) {
		t.Fatal(err)
	}
	assert.Equal(t, "base-workflow-template", wftmpl.Name)
	assert.Equal(t, "whalesay", tmpl.Name)

	// Get the template of template reference.
	tmplHolder = wfv1.Template{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "whalesay"}}
	ctx, tmpl, err = ctx.ResolveTemplate(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	wftmpl, ok = ctx.tmplBase.(*wfv1.WorkflowTemplate)
	if !assert.True(t, ok) {
		t.Fatal(err)
	}
	assert.Equal(t, "some-workflow-template", wftmpl.Name)
	assert.Equal(t, "whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	// Get the template of local nested template reference.
	tmplHolder = wfv1.Template{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "local-whalesay"}}
	ctx, tmpl, err = ctx.ResolveTemplate(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	wftmpl, ok = ctx.tmplBase.(*wfv1.WorkflowTemplate)
	if !assert.True(t, ok) {
		t.Fatal(err)
	}
	assert.Equal(t, "some-workflow-template", wftmpl.Name)
	assert.Equal(t, "local-whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	// Get the template of nested template reference.
	tmplHolder = wfv1.Template{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "another-whalesay"}}
	ctx, tmpl, err = ctx.ResolveTemplate(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	wftmpl, ok = ctx.tmplBase.(*wfv1.WorkflowTemplate)
	if !assert.True(t, ok) {
		t.Fatal(err)
	}
	assert.Equal(t, "another-workflow-template", wftmpl.Name)
	assert.Equal(t, "another-whalesay", tmpl.Name)
	assert.NotNil(t, tmpl.Container)

	// Get the template of template reference with arguments.
	tmplHolder = wfv1.Template{
		TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "whalesay-with-arguments"},
	}
	ctx, tmpl, err = ctx.ResolveTemplate(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	wftmpl, ok = ctx.tmplBase.(*wfv1.WorkflowTemplate)
	if !assert.True(t, ok) {
		t.Fatal(err)
	}
	assert.Equal(t, "some-workflow-template", wftmpl.Name)
	assert.Equal(t, "whalesay-with-arguments", tmpl.Name)
	assert.Equal(t, []string{"{{inputs.parameters.message}}-foo"}, tmpl.Container.Args)

	// Get the template of nested template reference with arguments.
	tmplHolder = wfv1.Template{
		TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "nested-whalesay-with-arguments"},
	}
	ctx, tmpl, err = ctx.ResolveTemplate(&tmplHolder)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	wftmpl, ok = ctx.tmplBase.(*wfv1.WorkflowTemplate)
	if !assert.True(t, ok) {
		t.Fatal(err)
	}
	assert.Equal(t, "some-workflow-template", wftmpl.Name)
	assert.Equal(t, "nested-whalesay-with-arguments", tmpl.Name)
	assert.Equal(t, []string{"{{inputs.parameters.message}}-bar-foo"}, tmpl.Container.Args)

	// Get the template of infinite loop template reference.
	tmplHolder = wfv1.Template{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "infinite-loop-whalesay"}}
	_, _, err = ctx.ResolveTemplate(&tmplHolder)
	assert.EqualError(t, err, "template reference exceeded max depth (10)")

	// Get the template of local infinite loop template.
	tmplHolder = wfv1.Template{TemplateRef: &wfv1.TemplateRef{Name: "some-workflow-template", Template: "infinite-local-loop-whalesay"}}
	_, _, err = ctx.ResolveTemplate(&tmplHolder)
	assert.EqualError(t, err, "template reference exceeded max depth (10)")
}
