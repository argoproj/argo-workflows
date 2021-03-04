package v1alpha1

import corev1 "k8s.io/api/core/v1"

type ContainerSetTemplate struct {
	Containers   []ContainerNode      `json:"containers" protobuf:"bytes,4,rep,name=containers"`
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty" protobuf:"bytes,3,rep,name=volumeMounts"`
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

type ContainerNode struct {
	corev1.Container `json:",inline" protobuf:"bytes,1,opt,name=container"`
	Dependencies     []string `json:"dependencies,omitempty" protobuf:"bytes,2,rep,name=dependencies"`
}
