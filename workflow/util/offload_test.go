package util

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argofake "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// fakeTemplateRepo is a hand-rolled implementor of the local TemplateRepo interface so the
// offload gate can be exercised without coupling the util package to persist/sqldb.
type fakeTemplateRepo struct {
	saved       bool
	savedUID    string
	savedNS     string
	savedTempls []wfv1.Template
}

func (f *fakeTemplateRepo) SaveTemplates(ctx context.Context, uid, namespace string, templates []wfv1.Template) error {
	f.saved = true
	f.savedUID = uid
	f.savedNS = namespace
	f.savedTempls = templates
	return nil
}

func TestShouldOffloadTemplates(t *testing.T) {
	tmpl1 := wfv1.Template{Name: "a", Container: &apiv1.Container{Image: "docker/whalesay:latest"}}
	tmpl2 := wfv1.Template{Name: "b", Container: &apiv1.Container{Image: "alpine:latest"}}

	tests := []struct {
		name      string
		templates []wfv1.Template
		minSize   int
		want      bool
	}{
		{"empty slice never offloads", nil, 0, false},
		{"empty slice never offloads even with minSize 1", nil, 1, false},
		{"two small templates under default threshold stay inline", []wfv1.Template{tmpl1, tmpl2}, 0, false},
		{"two small templates under default threshold stay inline (explicit 256KiB)", []wfv1.Template{tmpl1, tmpl2}, config.DefaultTemplateOffloadMinSize, false},
		{"two small templates offload when minSize is 1", []wfv1.Template{tmpl1, tmpl2}, 1, true},
		{"negative minSize behaves like the default", []wfv1.Template{tmpl1, tmpl2}, -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldOffloadTemplates(tt.templates, tt.minSize))
		})
	}
}

// twoSmallTemplates returns a couple of tiny templates whose serialized size is far below the
// default 256KiB threshold.
func twoSmallTemplates() []wfv1.Template {
	return []wfv1.Template{
		{Name: "a", Container: &apiv1.Container{Image: "docker/whalesay:latest"}},
		{Name: "b", Container: &apiv1.Container{Image: "alpine:latest"}},
	}
}

func TestCreateWorkflowWithOffload_SmallTemplatesStaysInline(t *testing.T) {
	repo := &fakeTemplateRepo{}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "small-wf", UID: "small-wf-uid", Namespace: "my-ns"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "whalesay",
			Templates:  twoSmallTemplates(),
		},
	}
	wfIf := argofake.NewClientset().ArgoprojV1alpha1().Workflows("my-ns")
	ctx := logging.TestContext(t.Context())

	created, err := CreateWorkflowWithOffload(ctx, wfIf, argofake.NewClientset(), repo, wf, 0)
	require.NoError(t, err)
	require.NotNil(t, created)

	// Templates must NOT be stripped and the DB must not be touched.
	assert.Len(t, created.Spec.Templates, 2, "small templates should remain inline")
	assert.Empty(t, created.Status.StoredTemplateSpecs, "no StoredTemplateSpecs when not offloaded")
	assert.False(t, repo.saved, "SaveTemplates must not be called for small templates")
}

func TestCreateWorkflowWithOffload_LargeTemplatesAreOffloaded(t *testing.T) {
	repo := &fakeTemplateRepo{}
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "big-wf", UID: "big-wf-uid", Namespace: "my-ns"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "whalesay",
			Templates:  twoSmallTemplates(),
		},
	}
	wfClientset := argofake.NewClientset()
	wfIf := wfClientset.ArgoprojV1alpha1().Workflows("my-ns")
	ctx := logging.TestContext(t.Context())

	// minOffloadSize 1 forces offloading regardless of the serialized size.
	created, err := CreateWorkflowWithOffload(ctx, wfIf, wfClientset, repo, wf, 1)
	require.NoError(t, err)
	require.NotNil(t, created)

	// Templates stripped from spec, DB round-tripped once with the originals, marker set.
	assert.Empty(t, created.Spec.Templates, "templates should be stripped after offloading")
	require.True(t, repo.saved, "SaveTemplates must be called when offloading")
	require.Len(t, repo.savedTempls, 2, "the original templates should be saved")
	assert.Equal(t, wf.Namespace, repo.savedNS)

	require.NotNil(t, created.Status.StoredTemplateSpecs, "StoredTemplateSpecs must be set")
	assert.Equal(t, string(created.UID), created.Status.StoredTemplateSpecs.UID)
	require.True(t, len(created.Status.StoredTemplateSpecs.Version) > 0 && created.Status.StoredTemplateSpecs.Version[:7] == "sha256:", "version must use the sha256: prefix")
}

func TestCreateWorkflowWithOffload_NilRepoKeepsTemplatesInline(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "no-repo-wf", UID: "no-repo-uid", Namespace: "my-ns"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "whalesay",
			Templates:  twoSmallTemplates(),
		},
	}
	wfClientset := argofake.NewClientset()
	wfIf := wfClientset.ArgoprojV1alpha1().Workflows("my-ns")
	ctx := logging.TestContext(t.Context())

	created, err := CreateWorkflowWithOffload(ctx, wfIf, wfClientset, nil, wf, 1)
	require.NoError(t, err)
	require.NotNil(t, created)

	// With no repo the offload path is entirely bypassed: templates untouched, no marker.
	assert.Len(t, created.Spec.Templates, 2, "templates must remain inline when no repo is configured")
	assert.Empty(t, created.Status.StoredTemplateSpecs)
}

func TestShouldOffloadTemplates_StructuralLimits(t *testing.T) {
	// An 800-task DAG whose templates serialize to only ~110KiB must still be
	// offloaded: the raw k8s create would fail the CRD dag.tasks maxItems(200)
	// that the offload stub path bypasses.
	dagTasks := make([]wfv1.DAGTask, 0, 800)
	for i := range 800 {
		dagTasks = append(dagTasks, wfv1.DAGTask{
			Name:     fmt.Sprintf("l0-%d", i),
			Template: "print",
		})
	}
	spec := []wfv1.Template{
		{Name: "main", DAG: &wfv1.DAGTemplate{Tasks: dagTasks}},
		{Name: "print", Container: &apiv1.Container{Image: "busybox"}},
	}
	require.True(t, ShouldOffloadTemplates(spec, config.DefaultTemplateOffloadMinSize),
		"structurally oversized specs (dag.tasks > 200) must offload regardless of serialized size")

	// >200 templates likewise
	many := make([]wfv1.Template, 0, 201)
	for i := range 201 {
		many = append(many, wfv1.Template{Name: fmt.Sprintf("t-%d", i), Container: &apiv1.Container{Image: "busybox"}})
	}
	require.True(t, ShouldOffloadTemplates(many, config.DefaultTemplateOffloadMinSize),
		"specs with >200 templates must offload (CRD spec.templates maxItems)")

	// 200 tasks and 200 templates remain inside the CRD limits -> size decides
	exactly200 := make([]wfv1.DAGTask, 0, 200)
	for i := range 200 {
		exactly200 = append(exactly200, wfv1.DAGTask{Name: fmt.Sprintf("l0-%d", i), Template: "print"})
	}
	inLimit := []wfv1.Template{
		{Name: "main", DAG: &wfv1.DAGTemplate{Tasks: exactly200}},
		{Name: "print", Container: &apiv1.Container{Image: "busybox"}},
	}
	require.False(t, ShouldOffloadTemplates(inLimit, config.DefaultTemplateOffloadMinSize),
		"specs within CRD limits follow the size gate")
}
