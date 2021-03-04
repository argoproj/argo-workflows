package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestContainerSetTemplate(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		x := &ContainerSetTemplate{}
		assert.Empty(t, x.GetGraph())
		assert.Empty(t, x.GetContainers())
		assert.False(t, x.HasSequencedContainers())
	})
	t.Run("Single", func(t *testing.T) {
		x := &ContainerSetTemplate{Containers: []ContainerNode{{}}}
		assert.Len(t, x.GetGraph(), 1)
		assert.Len(t, x.GetContainers(), 1)
		assert.False(t, x.HasSequencedContainers())
	})
	t.Run("Parallel", func(t *testing.T) {
		x := &ContainerSetTemplate{Containers: []ContainerNode{{}, {}}}
		assert.Len(t, x.GetGraph(), 2)
		assert.Len(t, x.GetContainers(), 2)
		assert.False(t, x.HasSequencedContainers())
	})
	t.Run("Graph", func(t *testing.T) {
		x := &ContainerSetTemplate{Containers: []ContainerNode{{Container: corev1.Container{Name: "a"}}, {Dependencies: []string{"a"}}}}
		assert.Len(t, x.GetGraph(), 2)
		assert.Len(t, x.GetContainers(), 2)
		assert.True(t, x.HasSequencedContainers())
		assert.True(t, x.HasContainerNamed("a"))
	})
}
