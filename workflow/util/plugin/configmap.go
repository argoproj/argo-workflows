package plugin

import (
	"fmt"
	"maps"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v4/pkg/plugins/spec"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func ToConfigMap(p *spec.Plugin) (*apiv1.ConfigMap, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(p.Spec.Sidecar.Container)
	if err != nil {
		return nil, err
	}
	cm := &apiv1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-executor-plugin", p.Name),
			Annotations: map[string]string{},
			Labels: map[string]string{
				common.LabelKeyConfigMapType: p.Kind,
			},
			Namespace: p.Namespace,
		},
		Data: map[string]string{
			"sidecar.automountServiceAccountToken": fmt.Sprint(p.Spec.Sidecar.AutomountServiceAccountToken),
			"sidecar.container":                    string(data),
		},
	}
	maps.Copy(cm.Annotations, p.Annotations)
	maps.Copy(cm.Labels, p.Labels)
	return cm, nil
}

func FromConfigMap(cm *apiv1.ConfigMap) (*spec.Plugin, error) {
	p := &spec.Plugin{
		TypeMeta: metav1.TypeMeta{
			Kind: cm.Labels[common.LabelKeyConfigMapType],
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        strings.TrimSuffix(cm.Name, "-executor-plugin"),
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
	}
	maps.Copy(p.Annotations, cm.Annotations)
	maps.Copy(p.Labels, cm.Labels)
	delete(p.Labels, common.LabelKeyConfigMapType)
	p.Spec.Sidecar.AutomountServiceAccountToken = cm.Data["sidecar.automountServiceAccountToken"] == "true"
	if err := yaml.UnmarshalStrict([]byte(cm.Data["sidecar.container"]), &p.Spec.Sidecar.Container); err != nil {
		return nil, err
	}
	return p, p.Validate()
}
