package slice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveString(t *testing.T) {
	t.Run("RemoveEmpty", func(t *testing.T) {
		slice := []string{}
		newSlice := RemoveString(slice, "1")
		assert.Equal(t, 0, len(newSlice))
		assert.NotContains(t, newSlice, "1")
	})

	t.Run("RemoveSingle", func(t *testing.T) {
		slice := []string{"1"}
		newSlice := RemoveString(slice, "3")
		assert.Equal(t, 1, len(newSlice))
		assert.Contains(t, newSlice, "1")
	})

	t.Run("RemoveSingleWithMatch", func(t *testing.T) {
		slice := []string{"1"}
		newSlice := RemoveString(slice, "1")
		assert.Equal(t, 0, len(newSlice))
		assert.NotContains(t, newSlice, "1")
	})

	t.Run("RemoveFirst", func(t *testing.T) {
		slice := []string{"1", "2", "3", "4", "5", "6"}
		newSlice := RemoveString(slice, "1")
		assert.Equal(t, 5, len(newSlice))
		assert.NotContains(t, newSlice, "1")
	})

	t.Run("RemoveMiddle", func(t *testing.T) {
		slice := []string{"1", "2", "3", "4", "5", "6"}
		newSlice := RemoveString(slice, "3")
		assert.Equal(t, 5, len(newSlice))
		assert.NotContains(t, newSlice, "3")
	})

	t.Run("RemoveLast", func(t *testing.T) {
		slice := []string{"1", "2", "3", "4", "5", "6"}
		newSlice := RemoveString(slice, "6")
		assert.Equal(t, 5, len(newSlice))
		assert.NotContains(t, newSlice, "6")
	})
}

func TestContainsString(t *testing.T) {
	slice := []string{"1", "2", "3", "4", "5", "6"}
	t.Run("FindFirst", func(t *testing.T) {
		assert.True(t, ContainsString(slice, "1"))
	})

	t.Run("FindMiddle", func(t *testing.T) {
		assert.True(t, ContainsString(slice, "4"))
	})

	t.Run("FindLast", func(t *testing.T) {
		assert.True(t, ContainsString(slice, "6"))
	})
	t.Run("NoFound", func(t *testing.T) {
		assert.False(t, ContainsString(slice, "7"))
	})
}
