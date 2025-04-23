package pod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestPodCleanupPatch(t *testing.T) {
	c := &Controller{}

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          map[string]string{common.LabelKeyCompleted: "false"},
			Finalizers:      []string{common.FinalizerPodStatus},
			ResourceVersion: "123456",
		},
	}

	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")

	// pod finalizer enabled, patch label
	patch, err := c.getPodCleanupPatch(pod, true)
	require.NoError(t, err)
	expected := `{"metadata":{"resourceVersion":"123456","finalizers":[],"labels":{"workflows.argoproj.io/completed":"true"}}}`
	assert.JSONEq(t, expected, string(patch))

	// pod finalizer enabled, do not patch label
	patch, err = c.getPodCleanupPatch(pod, false)
	require.NoError(t, err)
	expected = `{"metadata":{"resourceVersion":"123456","finalizers":[]}}`
	assert.JSONEq(t, expected, string(patch))

	// pod finalizer enabled, do not patch label, nil/empty finalizers
	podWithNilFinalizers := &apiv1.Pod{}
	patch, err = c.getPodCleanupPatch(podWithNilFinalizers, false)
	require.NoError(t, err)
	assert.Nil(t, patch)

	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")

	// pod finalizer disabled, patch both
	patch, err = c.getPodCleanupPatch(pod, true)
	require.NoError(t, err)
	expected = `{"metadata":{"labels":{"workflows.argoproj.io/completed":"true"}}}`
	assert.JSONEq(t, expected, string(patch))

	// pod finalizer disabled, do not patch label
	patch, err = c.getPodCleanupPatch(pod, false)
	require.NoError(t, err)
	assert.Nil(t, patch)
}
