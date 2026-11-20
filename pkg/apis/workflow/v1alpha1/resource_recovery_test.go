package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceRecoveryStatusProtobufRoundTrip(t *testing.T) {
	want := NodeStatus{
		ResourcePodRecoveries:    2,
		RecoveringResourcePodUID: "failed-resource-executor",
	}
	data, err := want.Marshal()
	require.NoError(t, err)
	var got NodeStatus
	require.NoError(t, got.Unmarshal(data))
	assert.Equal(t, want.ResourcePodRecoveries, got.ResourcePodRecoveries)
	assert.Equal(t, want.RecoveringResourcePodUID, got.RecoveringResourcePodUID)
}
