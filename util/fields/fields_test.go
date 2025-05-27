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

var complexWorkflow = `
metadata:
  name: complex-workflow
spec:
  entrypoint: main
status:
  phase: Succeeded
  startedAt: "2023-01-01T00:00:00Z"
  finishedAt: "2023-01-01T01:00:00Z"
  nodes:
    node-1:
      id: node-1
      name: node-1-name
      displayName: First Node
      type: Pod
      phase: Succeeded
      message: "All good"
      hostNodeName: "worker-1"
      templateName: template-a
      resourcesDuration:
        cpu: 100
        memory: 200
      inputs:
        parameters:
        - name: param1
          value: value1
      outputs:
        parameters:
        - name: output
          value: result1
    node-2:
      id: node-2
      name: node-2-name
      displayName: Second Node
      type: Pod
      phase: Failed
      message: "Something went wrong"
      hostNodeName: "worker-2"
      templateName: template-b
      resourcesDuration:
        cpu: 300
        memory: 400
      inputs:
        parameters:
        - name: param2
          value: value2
      outputs:
        parameters:
        - name: output
          value: result2
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

func TestCleanNodeMapFields(t *testing.T) {
	var wf, cleanWf wfv1.Workflow
	wfv1.MustUnmarshal([]byte(complexWorkflow), &wf)
	fields := "status.phase,status.startedAt,status.finishedAt,status.nodes," +
		"status.nodes.id,status.nodes.name,status.nodes.displayName," +
		"status.nodes.type,status.nodes.phase"

	ok, err := NewCleaner(fields).Clean(wf, &cleanWf)
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, wfv1.WorkflowSucceeded, cleanWf.Status.Phase)
	assert.NotEmpty(t, cleanWf.Status.StartedAt)
	assert.NotEmpty(t, cleanWf.Status.FinishedAt)
	assert.NotNil(t, cleanWf.Status.Nodes)
	assert.Len(t, cleanWf.Status.Nodes, 2)

	node1 := cleanWf.Status.Nodes["node-1"]
	assert.Equal(t, "node-1", node1.ID)
	assert.Equal(t, "node-1-name", node1.Name)
	assert.Equal(t, "First Node", node1.DisplayName)
	assert.Equal(t, wfv1.NodeSucceeded, node1.Phase)

	// These should be empty if filtering works correctly
	assert.Empty(t, node1.Message)
	assert.Empty(t, node1.HostNodeName)
	assert.Empty(t, node1.TemplateName)
	assert.Nil(t, node1.ResourcesDuration)
	assert.Nil(t, node1.Inputs)
	assert.Nil(t, node1.Outputs)
}

func TestCleanNodeMapFieldsExclude(t *testing.T) {
	var wf, cleanWf wfv1.Workflow
	wfv1.MustUnmarshal([]byte(complexWorkflow), &wf)

	// Define fields to include and specific node fields to exclude
	fields := "status.phase,status.startedAt,status.finishedAt,status.nodes," +
		"-status.nodes.message,-status.nodes.hostNodeName,-status.nodes.templateName," +
		"-status.nodes.resourcesDuration,-status.nodes.inputs,-status.nodes.outputs"

	ok, err := NewCleaner(fields).Clean(wf, &cleanWf)
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, wfv1.WorkflowSucceeded, cleanWf.Status.Phase)
	assert.NotEmpty(t, cleanWf.Status.StartedAt)
	assert.NotEmpty(t, cleanWf.Status.FinishedAt)
	assert.NotNil(t, cleanWf.Status.Nodes)

	node1 := cleanWf.Status.Nodes["node-1"]
	assert.Equal(t, "node-1", node1.ID)
	assert.Equal(t, "node-1-name", node1.Name)
	assert.Equal(t, "First Node", node1.DisplayName)
	assert.Equal(t, wfv1.NodeSucceeded, node1.Phase)

	// These should be empty if exclusion works correctly
	assert.Empty(t, node1.Message)
	assert.Empty(t, node1.HostNodeName)
	assert.Empty(t, node1.TemplateName)
	assert.Nil(t, node1.ResourcesDuration)
	assert.Nil(t, node1.Inputs)
	assert.Nil(t, node1.Outputs)
}
