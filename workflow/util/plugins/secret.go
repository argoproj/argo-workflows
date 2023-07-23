package plugin

import (
	"fmt"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/pkg/plugins/spec"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func ToSecret(p *spec.Plugin) (*apiv1.Secret, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(p.Spec.Sidecar.Container)
	if err != nil {
		return nil, err
	}
	secret := &apiv1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-executor-plugin", p.Name),
			Annotations: map[string]string{},
			Labels: map[string]string{
				common.LabelKeySecretType: p.Kind,
			},
			Namespace: p.Namespace,
		},
		StringData: map[string]string{
			"sidecar.automountServiceAccountToken": fmt.Sprint(p.Spec.Sidecar.AutomountServiceAccountToken),
			"sidecar.container":                    string(data),
		},
	}
	for k, v := range p.Annotations {
		secret.Annotations[k] = v
	}
	for k, v := range p.Labels {
		secret.Labels[k] = v
	}
	return secret, nil
}

func FromSecret(secret *apiv1.Secret) (*spec.Plugin, error) {
	p := &spec.Plugin{
		TypeMeta: metav1.TypeMeta{
			Kind: secret.Labels[common.LabelKeySecretType],
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        strings.TrimSuffix(secret.Name, "-executor-plugin"),
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
	}
	for k, v := range secret.Annotations {
		p.Annotations[k] = v
	}
	for k, v := range secret.Labels {
		p.Labels[k] = v
	}
	delete(p.Labels, common.LabelKeySecretType)
	p.Spec.Sidecar.AutomountServiceAccountToken = secret.StringData["sidecar.automountServiceAccountToken"] == "true"
	if err := yaml.UnmarshalStrict([]byte(secret.StringData["sidecar.container"]), &p.Spec.Sidecar.Container); err != nil {
		return nil, err
	}
	return p, p.Validate()
}
