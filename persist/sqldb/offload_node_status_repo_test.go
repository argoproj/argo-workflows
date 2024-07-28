package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

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
