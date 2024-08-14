package slice

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveString(t *testing.T) {
	t.Run("RemoveEmpty", func(t *testing.T) {
		slice := []string{}
		newSlice := RemoveString(slice, "1")
		require.Empty(t, newSlice)
		require.NotContains(t, newSlice, "1")
	})

	t.Run("RemoveSingle", func(t *testing.T) {
		slice := []string{"1"}
		newSlice := RemoveString(slice, "3")
		require.Len(t, newSlice, 1)
		require.Contains(t, newSlice, "1")
	})

	t.Run("RemoveSingleWithMatch", func(t *testing.T) {
		slice := []string{"1"}
		newSlice := RemoveString(slice, "1")
		require.Empty(t, newSlice)
		require.NotContains(t, newSlice, "1")
	})

	t.Run("RemoveFirst", func(t *testing.T) {
		slice := []string{"1", "2", "3", "4", "5", "6"}
		newSlice := RemoveString(slice, "1")
		require.Len(t, newSlice, 5)
		require.NotContains(t, newSlice, "1")
	})

	t.Run("RemoveMiddle", func(t *testing.T) {
		slice := []string{"1", "2", "3", "4", "5", "6"}
		newSlice := RemoveString(slice, "3")
		require.Len(t, newSlice, 5)
		require.NotContains(t, newSlice, "3")
	})

	t.Run("RemoveLast", func(t *testing.T) {
		slice := []string{"1", "2", "3", "4", "5", "6"}
		newSlice := RemoveString(slice, "6")
		require.Len(t, newSlice, 5)
		require.NotContains(t, newSlice, "6")
	})
}

func TestContainsString(t *testing.T) {
	slice := []string{"1", "2", "3", "4", "5", "6"}
	t.Run("FindFirst", func(t *testing.T) {
		require.True(t, ContainsString(slice, "1"))
	})

	t.Run("FindMiddle", func(t *testing.T) {
		require.True(t, ContainsString(slice, "4"))
	})

	t.Run("FindLast", func(t *testing.T) {
		require.True(t, ContainsString(slice, "6"))
	})
	t.Run("NoFound", func(t *testing.T) {
		require.False(t, ContainsString(slice, "7"))
	})
}
