package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/workflow/sync/mocks"
)

func TestChainThrottler(t *testing.T) {
	m := &mocks.Throttler{}
	m.On("Add", "foo", int32(1), time.Time{}).Return()
	m.On("Admit", "foo").Return(false)
	m.On("Remove", "foo").Return()

	c := ChainThrottler{m}
	c.Add("foo", 1, time.Time{})
	require.False(t, c.Admit("foo"))
	c.Remove("foo")

	require.True(t, ChainThrottler{}.Admit("foo"))
}
