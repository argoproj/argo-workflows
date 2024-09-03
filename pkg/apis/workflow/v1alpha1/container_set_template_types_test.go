package v1alpha1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/yaml"
)

func validateContainerSetTemplate(yamlStr string) error {
	var cst ContainerSetTemplate
	err := yaml.Unmarshal([]byte(yamlStr), &cst)
	if err != nil {
		panic(err)
	}
	return cst.Validate()
}

func TestContainerSetGetRetryStrategy(t *testing.T) {
	t.Run("RetriesOnly", func(t *testing.T) {
		retries := intstr.FromInt(100)
		set := ContainerSetTemplate{
			RetryStrategy: &ContainerSetRetryStrategy{
				Retries: &retries,
			},
		}
		strategy, err := set.GetRetryStrategy()
		require.NoError(t, err)
		assert.Equal(t, wait.Backoff{Steps: 100}, strategy)
	})

	t.Run("DurationSet", func(t *testing.T) {
		retries := intstr.FromInt(100)
		duration := "20s"
		set := &ContainerSetTemplate{
			RetryStrategy: &ContainerSetRetryStrategy{
				Retries:  &retries,
				Duration: duration,
			},
		}
		strategy, err := set.GetRetryStrategy()
		require.NoError(t, err)
		assert.Equal(t, wait.Backoff{
			Steps:    100,
			Duration: time.Duration(20 * time.Second),
		}, strategy)
	})
}

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

func TestInvalidContainerSetEmpty(t *testing.T) {
	invalidContainerSetEmpty := `
volumeMounts:
  - name: workspace
    mountPath: /workspace
`
	err := validateContainerSetTemplate(invalidContainerSetEmpty)
	require.ErrorContains(t, err, "containers must have at least one container")
}

func TestInvalidContainerSetDuplicateVolumeMounting(t *testing.T) {
	invalidContainerSetDuplicateNames := `
volumeMounts:
  - name: workspace
    mountPath: /workspace
  - name: workspace2
    mountPath: /workspace
containers:
  - name: a
    image: argoproj/argosay:v2
`
	err := validateContainerSetTemplate(invalidContainerSetDuplicateNames)
	require.ErrorContains(t, err, "volumeMounts[1].mountPath '/workspace' already mounted in volumeMounts.workspace")
}

func TestInvalidContainerSetDuplicateNames(t *testing.T) {
	invalidContainerSetDuplicateNames := `
volumeMounts:
  - name: workspace
    mountPath: /workspace
containers:
  - name: a
    image: argoproj/argosay:v2
  - name: a
    image: argoproj/argosay:v2
`
	err := validateContainerSetTemplate(invalidContainerSetDuplicateNames)
	require.ErrorContains(t, err, "containers[1].name 'a' is not unique")

}

func TestInvalidContainerSetDependencyNotFound(t *testing.T) {
	invalidContainerSetDependencyNotFound := `
volumeMounts:
  - name: workspace
    mountPath: /workspace
containers:
  - name: a
    image: argoproj/argosay:v2
  - name: b
    image: argoproj/argosay:v2
    dependencies:
      - c
`
	err := validateContainerSetTemplate(invalidContainerSetDependencyNotFound)
	require.ErrorContains(t, err, "containers.b dependency 'c' not defined")
}

func TestInvalidContainerSetDependencyCycle(t *testing.T) {
	invalidContainerSetDependencyCycle := `
volumeMounts:
  - name: workspace
    mountPath: /workspace
containers:
  - name: a
    image: argoproj/argosay:v2
    dependencies:
      - b
  - name: b
    image: argoproj/argosay:v2
    dependencies:
      - a
`
	err := validateContainerSetTemplate(invalidContainerSetDependencyCycle)
	require.ErrorContains(t, err, "containers dependency cycle detected: b->a->b")
}

func TestValidContainerSet(t *testing.T) {
	validContainerSet := `
volumeMounts:
  - name: workspace
    mountPath: /workspace
containers:
  - name: a
    image: argoproj/argosay:v2
  - name: b
    image: argoproj/argosay:v2
    dependencies:
      - a
  - name: c
    image: argoproj/argosay:v2
    dependencies:
      - a
  - name: d
    image: argoproj/argosay:v2
    dependencies:
      - b
      - c
`
	err := validateContainerSetTemplate(validContainerSet)
	require.NoError(t, err)
}
