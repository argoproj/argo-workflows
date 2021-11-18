package plugin

import (
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type plugin struct {
	Kind     string            `json:"kind"`
	Metadata corev1.ObjectMeta `json:"metadata"`
	Spec     struct {
		Description string          `json:"description"`
		Address     string          `json:"address"`
		Container   apiv1.Container `json:"container"`
	} `json:"spec"`
}

func (p plugin) ShortKind() interface{} {
	switch p.Kind {
	case "ExecutorPlugin":
		return "executor"
	default:
		return "controller"
	}
}
