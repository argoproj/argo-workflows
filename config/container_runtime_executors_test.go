package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestContainerRuntimeExecutors(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		x := ContainerRuntimeExecutors{}
		e, err := x.Select(labels.Set{})
		assert.NoError(t, err)
		assert.Empty(t, e)
	})
	t.Run("Select", func(t *testing.T) {
		x := ContainerRuntimeExecutors{
			{
				Name: "foo",
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{"bar": ""},
				},
			},
		}
		e, err := x.Select(labels.Set(map[string]string{"bar": ""}))
		assert.NoError(t, err)
		assert.Equal(t, "foo", e)
	})
	t.Run("Error", func(t *testing.T) {
		x := ContainerRuntimeExecutors{
			{
				Name: "foo",
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{"!": "!"},
				},
			},
		}
		_, err := x.Select(labels.Set(map[string]string{"bar": ""}))
		assert.Error(t, err)
	})
}
