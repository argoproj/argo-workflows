package packer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestDefault(t *testing.T) {
	assert.Equal(t, 1024*1024, getMaxWorkflowSize())
}

func TestDecompressWorkflow(t *testing.T) {
	cleanup := SetMaxWorkflowSize(230)
	defer cleanup()
	ctx := logging.TestContext(t.Context())

	t.Run("SmallWorkflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}},
			},
		}
		err := CompressWorkflowIfNeeded(ctx, wf)
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.NotEmpty(t, wf.Status.Nodes)
		assert.Empty(t, wf.Status.CompressedNodes)

		err = DecompressWorkflow(ctx, wf)
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.NotEmpty(t, wf.Status.Nodes)
		assert.Empty(t, wf.Status.CompressedNodes)
	})
	t.Run("LargeWorkflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}, "bar": wfv1.NodeStatus{}},
			},
		}
		err := CompressWorkflowIfNeeded(ctx, wf)
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.Empty(t, wf.Status.Nodes)
		assert.NotEmpty(t, wf.Status.CompressedNodes)

		err = DecompressWorkflow(ctx, wf)
		require.NoError(t, err)
		assert.NotNil(t, wf)
		assert.NotEmpty(t, wf.Status.Nodes)
		assert.Empty(t, wf.Status.CompressedNodes)
	})
	t.Run("TooLargeToCompressWorkflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{Entrypoint: "main"},
			Status: wfv1.WorkflowStatus{
				Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{}, "bar": wfv1.NodeStatus{}, "baz": wfv1.NodeStatus{}, "qux": wfv1.NodeStatus{}},
			},
		}
		err := CompressWorkflowIfNeeded(ctx, wf)
		require.Error(t, err)
		assert.True(t, IsTooLargeError(err))
		// if too large, we want the original back please
		assert.NotNil(t, wf)
		assert.NotEmpty(t, wf.Status.Nodes)
		assert.Empty(t, wf.Status.CompressedNodes)
	})
}

func TestCompressWorkflowTaskSetSpec(t *testing.T) {
	cleanup := SetMaxWorkflowSize(275)
	defer cleanup()
	ctx := logging.TestContext(t.Context())

	t.Run("LargeTaskSet", func(t *testing.T) {
		pattern := &wfv1.WorkflowTaskSetSpec{
			Tasks: map[string]wfv1.Template{
				"foo": {
					Name: "foo-" + strings.Repeat("x", 200),
					Metadata: wfv1.Metadata{
						Annotations: map[string]string{
							"big":    strings.Repeat("a", 1000),
							"bigger": strings.Repeat("b", 1000),
						},
					},
					Container: &apiv1.Container{
						Image: "alpine",
					},
				},
			},
		}

		spec := pattern.DeepCopy()

		err := CompressWorkflowTaskSetSpec(ctx, spec)
		require.NoError(t, err)

		assert.NotNil(t, spec)
		assert.Len(t, spec.Tasks, 1)

		for name, task := range spec.Tasks {
			assert.NotEmpty(
				t,
				task.CompressedTemplate,
				"empty compressed template field after large task: %s compression", name,
			)

			expectedCompressedTemplate := wfv1.Template{
				CompressedTemplate: task.CompressedTemplate,
			}

			assert.Equal(t, expectedCompressedTemplate, task)
		}

		err = DecompressWorkflowTaskSetSpec(ctx, spec)
		require.NoError(t, err)

		assert.Len(t, spec.Tasks, 1)

		for name, task := range spec.Tasks {
			assert.Empty(
				t,
				task.CompressedTemplate,
				"not empty compressed template field after large task: %s decompression", name,
			)
			assert.Equal(t, task, pattern.Tasks[name])
		}
	})

	t.Run("TooLargeToCompressTaskSet", func(t *testing.T) {
		pattern := &wfv1.WorkflowTaskSetSpec{
			Tasks: map[string]wfv1.Template{
				"foo": {
					Name: "foo-" + strings.Repeat("x", 200),
					Metadata: wfv1.Metadata{
						Annotations: map[string]string{
							"big":    strings.Repeat("a", 1000),
							"bigger": strings.Repeat("b", 1000),
						},
					},
					Container: &apiv1.Container{
						Image: "alpine",
						Args: []string{
							strings.Repeat("arg1-", 200),
							strings.Repeat("arg2-", 200),
						},
					},
				},
				"bar": {
					Name: "bar-" + strings.Repeat("y", 200),
					Metadata: wfv1.Metadata{
						Annotations: map[string]string{
							"x": strings.Repeat("x", 1000),
						},
					},
					Container: &apiv1.Container{
						Image: "alpine",
						Args: []string{
							strings.Repeat("bar-", 400),
						},
					},
					Inputs: wfv1.Inputs{
						Parameters: []wfv1.Parameter{
							{
								Name:  "foo-param",
								Value: wfv1.AnyStringPtr(strings.Repeat("bar-", 4000)),
							},
						},
					},
					Outputs: wfv1.Outputs{
						Parameters: []wfv1.Parameter{
							{
								Name:  "foo-param",
								Value: wfv1.AnyStringPtr(strings.Repeat("bar-", 4000)),
							},
						},
					},
				},
			},
		}

		spec := pattern.DeepCopy()

		err := CompressWorkflowTaskSetSpec(ctx, spec)

		require.Error(t, err)
		assert.True(t, IsTooLargeTaskSetSpecError(err))

		// if too large, we want the original back
		assert.Len(t, spec.Tasks, 2)

		for name, task := range spec.Tasks {
			assert.Empty(
				t,
				task.CompressedTemplate,
				"not empty compressed template field after too large task: %s failed compression", name,
			)
			assert.Equal(t, task, pattern.Tasks[name])
		}
	})
}

func TestCompressWorkflowTaskSetStatus(t *testing.T) {
	cleanup := SetMaxWorkflowSize(250)
	defer cleanup()
	ctx := logging.TestContext(t.Context())

	t.Run("LargeNodeSet", func(t *testing.T) {
		r1000 := strings.Repeat("r", 1000)
		pattern := &wfv1.WorkflowTaskSetStatus{
			Nodes: map[string]wfv1.NodeResult{
				"foo": {
					Phase:   wfv1.NodeRunning,
					Message: strings.Repeat("m", 500),
					Outputs: &wfv1.Outputs{
						Result: &r1000,
					},
					Progress: wfv1.ProgressZero,
				},
			},
		}

		status := pattern.DeepCopy()

		err := CompressWorkflowTaskSetStatus(ctx, status)
		require.NoError(t, err)

		assert.NotNil(t, status)
		assert.Len(t, status.Nodes, 1)

		for name, node := range status.Nodes {
			assert.NotEmpty(
				t,
				node.CompressedNode,
				"empty compressed node after compression: %s", name,
			)

			expected := wfv1.NodeResult{
				CompressedNode: node.CompressedNode,
			}

			assert.Equal(t, expected, node)
		}

		err = DecompressWorkflowTaskSetStatus(ctx, status)
		require.NoError(t, err)

		assert.Len(t, status.Nodes, 1)

		for name, node := range status.Nodes {
			assert.Empty(
				t,
				node.CompressedNode,
				"compressedNode not empty after decompression: %s", name,
			)

			assert.Equal(t, pattern.Nodes[name], node)
		}
	})

	t.Run("TooLargeToCompressNodeSet", func(t *testing.T) {
		r2000 := strings.Repeat("r", 2000)
		y2000 := strings.Repeat("y", 2000)
		pattern := &wfv1.WorkflowTaskSetStatus{
			Nodes: map[string]wfv1.NodeResult{
				"foo": {
					Phase:   wfv1.NodeRunning,
					Message: strings.Repeat("m", 2000),
					Outputs: &wfv1.Outputs{
						Result: &r2000,
					},
				},
				"bar": {
					Phase:   wfv1.NodeFailed,
					Message: strings.Repeat("x", 2000),
					Outputs: &wfv1.Outputs{
						Result: &y2000,
					},
				},
			},
		}

		status := pattern.DeepCopy()

		err := CompressWorkflowTaskSetStatus(ctx, status)

		require.Error(t, err)
		assert.True(t, IsTooLargeTaskSetSpecError(err))

		// при фейле состояние должно восстановиться
		assert.Len(t, status.Nodes, 2)

		for name, node := range status.Nodes {
			assert.Empty(
				t,
				node.CompressedNode,
				"compressedNode should be empty after failed compression: %s", name,
			)

			assert.Equal(t, pattern.Nodes[name], node)
		}
	})
}
