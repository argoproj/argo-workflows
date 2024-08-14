package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/pointer"
)

func TestNodeEvents_IsEnabled(t *testing.T) {
	require.True(t, NodeEvents{}.IsEnabled())
	require.False(t, NodeEvents{Enabled: pointer.Bool(false)}.IsEnabled())
	require.True(t, NodeEvents{Enabled: pointer.Bool(true)}.IsEnabled())
}
