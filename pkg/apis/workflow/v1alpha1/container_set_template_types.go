package v1alpha1

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

type ContainerSetTemplate struct {
	Containers   []ContainerNode      `json:"containers" protobuf:"bytes,4,rep,name=containers"`
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty" protobuf:"bytes,3,rep,name=volumeMounts"`
	// RetryStrategy describes how to retry container nodes if the container set fails.
	// Note that this works differently from the template-level `retryStrategy` as it is a process-level retry that does not create new Pods or containers.
	RetryStrategy *ContainerSetRetryStrategy `json:"retryStrategy,omitempty" protobuf:"bytes,5,opt,name=retryStrategy"`
}

// ContainerSetRetryStrategy provides controls on how to retry a container set
type ContainerSetRetryStrategy struct {
	// Duration is the time between each retry, examples values are "300ms", "1s" or "5m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	Duration string `json:"duration,omitempty" protobuf:"bytes,1,opt,name=duration"`
	// Retries is the maximum number of retry attempts for each container. It does not include the
	// first, original attempt; the maximum number of total attempts will be `retries + 1`.
	Retries *intstr.IntOrString `json:"retries" protobuf:"bytes,2,rep,name=retries"`
}

func (t *ContainerSetTemplate) GetRetryStrategy() (wait.Backoff, error) {
	if t == nil || t.RetryStrategy == nil || t.RetryStrategy.Retries == nil {
		return wait.Backoff{Steps: 1}, nil
	}

	backoff := wait.Backoff{Steps: t.RetryStrategy.Retries.IntValue()}

	if t.RetryStrategy.Duration == "" {
		return backoff, nil
	}

	baseDuration, err := time.ParseDuration(t.RetryStrategy.Duration)
	if err != nil {
		return wait.Backoff{}, err
	}

	if baseDuration < time.Duration(0) {
		return wait.Backoff{}, fmt.Errorf("duration has to be positive, current duration: %v ", baseDuration)
	}

	backoff.Duration = baseDuration
	return backoff, nil
}

func (in *ContainerSetTemplate) GetContainers() []corev1.Container {
	var ctrs []corev1.Container
	for _, t := range in.GetGraph() {
		c := t.Container
		c.VolumeMounts = append(c.VolumeMounts, in.VolumeMounts...)
		ctrs = append(ctrs, c)
	}
	return ctrs
}

func (in *ContainerSetTemplate) HasContainerNamed(n string) bool {
	for _, c := range in.GetContainers() {
		if n == c.Name {
			return true
		}
	}
	return false
}

func (in *ContainerSetTemplate) GetGraph() []ContainerNode {
	if in == nil {
		return nil
	}
	return in.Containers
}

func (in *ContainerSetTemplate) HasSequencedContainers() bool {
	for _, n := range in.GetGraph() {
		if len(n.Dependencies) > 0 {
			return true
		}
	}
	return false
}

// Validate checks if the ContainerSetTemplate is valid
func (in *ContainerSetTemplate) Validate() error {
	if len(in.Containers) == 0 {
		return fmt.Errorf("containers must have at least one container")
	}

	names := make([]string, 0)
	for _, ctr := range in.Containers {
		names = append(names, ctr.Name)
	}
	err := validateWorkflowFieldNames(names, false)
	if err != nil {
		return fmt.Errorf("containers%s", err.Error())
	}

	// Ensure there are no collisions with volume mountPaths and artifact load paths
	mountPaths := make(map[string]string)
	for i, volMount := range in.VolumeMounts {
		if prev, ok := mountPaths[volMount.MountPath]; ok {
			return fmt.Errorf("volumeMounts[%d].mountPath '%s' already mounted in %s", i, volMount.MountPath, prev)
		}
		mountPaths[volMount.MountPath] = fmt.Sprintf("volumeMounts.%s", volMount.Name)
	}

	// Ensure the dependencies are defined
	nameToContainer := make(map[string]ContainerNode)
	for _, ctr := range in.Containers {
		nameToContainer[ctr.Name] = ctr
	}
	for _, ctr := range in.Containers {
		for _, depName := range ctr.Dependencies {
			_, ok := nameToContainer[depName]
			if !ok {
				return fmt.Errorf("containers.%s dependency '%s' not defined", ctr.Name, depName)
			}
		}
	}

	// Ensure there is no dependency cycle
	depGraph := make(map[string][]string)
	for _, ctr := range in.Containers {
		depGraph[ctr.Name] = append(depGraph[ctr.Name], ctr.Dependencies...)
	}
	err = validateNoCycles(depGraph)
	if err != nil {
		return fmt.Errorf("containers %s", err.Error())
	}
	return nil
}

type ContainerNode struct {
	corev1.Container `json:",inline" protobuf:"bytes,1,opt,name=container"`
	Dependencies     []string `json:"dependencies,omitempty" protobuf:"bytes,2,rep,name=dependencies"`
}
