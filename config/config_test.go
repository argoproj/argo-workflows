package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestDatabaseConfig(t *testing.T) {
	assert.Equal(t, "my-host", DatabaseConfig{Host: "my-host"}.GetHostname())
	assert.Equal(t, "my-host:1234", DatabaseConfig{Host: "my-host", Port: 1234}.GetHostname())
}

func TestContainerRuntimeExecutor(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		c := Config{ContainerRuntimeExecutor: "foo"}
		executor, err := c.GetContainerRuntimeExecutor(labels.Set{})
		assert.NoError(t, err)
		assert.Equal(t, "foo", executor)
	})
	t.Run("Error", func(t *testing.T) {
		c := Config{ContainerRuntimeExecutor: "foo", ContainerRuntimeExecutors: ContainerRuntimeExecutors{
			{Name: "bar", Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"!": "!"},
			}},
		}}
		_, err := c.GetContainerRuntimeExecutor(labels.Set{})
		assert.Error(t, err)
	})
	t.Run("NoError", func(t *testing.T) {
		c := Config{ContainerRuntimeExecutor: "foo", ContainerRuntimeExecutors: ContainerRuntimeExecutors{
			{Name: "bar", Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"baz": "qux"},
			}},
		}}
		executor, err := c.GetContainerRuntimeExecutor(labels.Set(map[string]string{"baz": "qux"}))
		assert.NoError(t, err)
		assert.Equal(t, "bar", executor)
	})
}
