package config

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type ContainerRuntimeExecutors []ContainerRuntimeExecutor

// select the correct executor to use
// this may return an empty string of there is not executor found
func (e ContainerRuntimeExecutors) Select(labels labels.Labels) (string, error) {
	for _, c := range e {
		ok, err := c.Matches(labels)
		if err != nil {
			return "", err
		}
		if ok {
			return c.Name, nil
		}
	}
	return "", nil
}

type ContainerRuntimeExecutor struct {
	Name     string               `json:"name"`
	Selector metav1.LabelSelector `json:"selector"`
}

func (e ContainerRuntimeExecutor) Matches(labels labels.Labels) (bool, error) {
	x, err := metav1.LabelSelectorAsSelector(&e.Selector)
	if err != nil {
		return false, err
	}
	return x.Matches(labels), nil
}
