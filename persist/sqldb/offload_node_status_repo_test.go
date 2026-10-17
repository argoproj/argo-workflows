package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Test_ListWithoutKeys guards the empty-keys early return. Without it the generated condition
// is empty, which would widen the query back to the whole namespace. A zero-value repo has no
// session, so anything past the guard panics rather than quietly querying.
func Test_ListWithoutKeys(t *testing.T) {
	got, err := (&nodeOffloadRepo{}).List(logging.TestContext(t.Context()), "argo", nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func Test_nodeStatusVersion(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		marshalled, version, err := nodeStatusVersion(nil)
		require.NoError(t, err)
		assert.NotEmpty(t, marshalled)
		assert.Equal(t, "fnv:784127654", version)
	})
	t.Run("NonEmpty", func(t *testing.T) {
		marshalled, version, err := nodeStatusVersion(wfv1.Nodes{"my-node": wfv1.NodeStatus{}})
		require.NoError(t, err)
		assert.NotEmpty(t, marshalled)
		assert.Equal(t, "fnv:2308444803", version)
	})
}
