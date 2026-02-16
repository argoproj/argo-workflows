package fields

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var sampleWorkflow = `
metadata:
  name: hello-world-qgpxz
spec:
  entrypoint: whalesay
status:
  nodes:
    hello-world-qgpxz:
      displayName: hello-world-qgpxz
  phase: Succeeded
`

func TestCleaner_WillExclude(t *testing.T) {
	t.Run("Noop", func(t *testing.T) {
		assert.False(t, NewCleaner("").WillExclude("foo"), "special case - keep everything")
	})
	t.Run("Default", func(t *testing.T) {
		assert.False(t, NewCleaner("foo").WillExclude("foo"))
		assert.False(t, NewCleaner("foo").WillExclude("foo.bar"))
		assert.True(t, NewCleaner("foo").WillExclude("bar"))
		assert.False(t, NewCleaner("foo.bar.baz").WillExclude("foo.bar"))
	})
	t.Run("Exclude", func(t *testing.T) {
		assert.True(t, NewCleaner("-foo").WillExclude("foo"))
		assert.True(t, NewCleaner("-foo").WillExclude("foo.bar"))
		assert.False(t, NewCleaner("-foo").WillExclude("bar"))
		assert.False(t, NewCleaner("-foo").WillExclude("bar.baz"))
	})
}

func TestCleaner_WithPrefix(t *testing.T) {
	cleaner := NewCleaner("result.object.status").WithoutPrefix("result.object.")
	assert.False(t, cleaner.fields["result.object.status"])
	assert.True(t, cleaner.fields["status"])
}

func TestCleanNoop(t *testing.T) {
	var wf, cleanWf wfv1.Workflow
	ok, err := NewCleaner("").Clean(wf, cleanWf)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestCleanFields(t *testing.T) {
	var wf, cleanWf wfv1.Workflow
	wfv1.MustUnmarshal([]byte(sampleWorkflow), &wf)
	ok, err := NewCleaner("status.phase,metadata.name,spec.entrypoint").Clean(wf, &cleanWf)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, wfv1.WorkflowSucceeded, cleanWf.Status.Phase)
	assert.Equal(t, "whalesay", cleanWf.Spec.Entrypoint)
	assert.Equal(t, "hello-world-qgpxz", cleanWf.Name)
	assert.Nil(t, cleanWf.Status.Nodes)
}

func TestCleanFieldsExclude(t *testing.T) {
	var wf, cleanWf wfv1.Workflow
	wfv1.MustUnmarshal([]byte(sampleWorkflow), &wf)
	ok, err := NewCleaner("-status.phase,metadata.name,spec.entrypoint").Clean(wf, &cleanWf)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Empty(t, cleanWf.Status.Phase)
	assert.Empty(t, cleanWf.Spec.Entrypoint)
	assert.Empty(t, cleanWf.Name)
	assert.NotNil(t, cleanWf.Status.Nodes)
}
