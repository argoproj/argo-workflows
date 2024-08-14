package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProgress(t *testing.T) {
	t.Run("ParseProgress", func(t *testing.T) {
		_, ok := ParseProgress("")
		require.False(t, ok)
		progress, ok := ParseProgress("0/1")
		require.True(t, ok)
		require.Equal(t, Progress("0/1"), progress)
	})
	t.Run("IsValid", func(t *testing.T) {
		require.False(t, Progress("").IsValid())
		require.False(t, Progress("/0").IsValid())
		require.False(t, Progress("0/").IsValid())
		require.False(t, Progress("0/0").IsValid())
		require.False(t, Progress("1/0").IsValid())
		require.True(t, Progress("0/1").IsValid())
	})
	t.Run("Add", func(t *testing.T) {
		require.Equal(t, Progress("1/2"), Progress("0/0").Add("1/2"))
	})
	t.Run("Complete", func(t *testing.T) {
		require.Equal(t, Progress("100/100"), Progress("0/100").Complete())
	})
}
