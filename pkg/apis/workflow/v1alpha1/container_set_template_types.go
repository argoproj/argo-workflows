package v1alpha1

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
)

type ContainerSetTemplate struct {
	Containers    []ContainerNode            `json:"containers" protobuf:"bytes,4,rep,name=containers"`
	VolumeMounts  []corev1.VolumeMount       `json:"volumeMounts,omitempty" protobuf:"bytes,3,rep,name=volumeMounts"`
	RetryStrategy *ContainerSetRetryStrategy `json:"retryStrategy,omitempty" protobuf:"bytes,5,opt,name=retryStrategy"`
}

type ContainerSetRetryStrategy struct {
	// The initial duration.
	Duration time.Duration `protobuf:"varint,1,opt,name=duration,casttype=time.Duration"`
	// Duration is multiplied by factor each iteration, if factor is not zero
	// and the limits imposed by Steps and Cap have not been reached.
	// Should not be negative.
	// The jitter does not contribute to the updates to the duration parameter.
	Factor float64 `protobuf:"fixed64,2,opt,name=factor"`
	// The sleep at each iteration is the duration plus an additional
	// amount chosen uniformly at random from the interval between
	// zero and `jitter*duration`.
	Jitter float64 `protobuf:"fixed64,3,opt,name=jitter"`
	// The remaining number of iterations in which the duration
	// parameter may change (but progress can be stopped earlier by
	// hitting the cap). If not positive, the duration is not
	// changed. Used for exponential backoff in combination with
	// Factor and Cap.
	Steps int `protobuf:"varint,4,opt,name=steps"`
	// A limit on revised values of the duration parameter. If a
	// multiplication by the factor parameter would make the duration
	// exceed the cap then the duration is set to the cap and the
	// steps parameter is set to zero.
	Cap time.Duration `protobuf:"varint,5,opt,name=cap,casttype=time.Duration"`
}

func (t *ContainerSetTemplate) GetRetryStrategy() (wait.Backoff, error) {
	if t == nil || t.RetryStrategy == nil || t.RetryStrategy.Limit == nil {
		return wait.Backoff{Steps: 1}, nil
	}

	retry := t.RetryStrategy
	backoff := wait.Backoff{Steps: retry.Limit.IntValue()}
	if retry.Backoff == nil {
		return backoff, nil
	}

	if retry.Backoff.Duration != "" {
		duration, err := time.ParseDuration(retry.Backoff.Duration)
		if err != nil {
			return wait.Backoff{}, fmt.Errorf("failed to parse retry duration: %w", err)
		}
		backoff.Duration = duration
	}

	if retry.Backoff.MaxDuration != "" {
		cap, err := time.ParseDuration(retry.Backoff.MaxDuration)
		if err != nil {
			return wait.Backoff{}, fmt.Errorf("failed to parse max duration: %w", err)
		}
		backoff.Cap = cap
	}

	if retry.Backoff.Factor != nil {
		backoff.Factor = float64(retry.Backoff.Factor.IntVal)
	}

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
