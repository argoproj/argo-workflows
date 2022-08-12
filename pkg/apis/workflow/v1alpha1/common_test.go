package v1alpha1

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareByPriority(t *testing.T) {

	t.Run("SortingByPriority", func(t *testing.T) {
		stepGroup := []WorkflowStep{{Name: "A", Priority: 10}, {Name: "B", Priority: 1000}, {Name: "C", Priority: 100}, {Name: "D", Priority: 10000}}

		sort.Slice(stepGroup, func(i, j int) bool {
			return CompareByPriority(&stepGroup[i], &stepGroup[j])
		})

		assert.Equal(t, "D", stepGroup[0].Name)
		assert.Equal(t, "B", stepGroup[1].Name)
		assert.Equal(t, "C", stepGroup[2].Name)
		assert.Equal(t, "A", stepGroup[3].Name)
	})

	t.Run("SortingByName", func(t *testing.T) {
		stepGroup := []WorkflowStep{{Name: "Banana"}, {Name: "Durian"}, {Name: "Apple"}, {Name: "Cherry"}}

		sort.Slice(stepGroup, func(i, j int) bool {
			return CompareByPriority(&stepGroup[i], &stepGroup[j])
		})

		assert.Equal(t, "Durian", stepGroup[0].Name)
		assert.Equal(t, "Cherry", stepGroup[1].Name)
		assert.Equal(t, "Banana", stepGroup[2].Name)
		assert.Equal(t, "Apple", stepGroup[3].Name)
	})
}
