package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/pkg/plugins/spec"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestToConfigMap(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		_, err := ToConfigMap(&spec.Plugin{})
		require.EqualError(t, err, "sidecar is invalid: at least one port is mandatory")
	})
	t.Run("Valid", func(t *testing.T) {
		cm, err := ToConfigMap(&spec.Plugin{
			TypeMeta: metav1.TypeMeta{
				Kind: common.LabelValueTypeConfigMapExecutorPlugin,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-plug",
				Annotations: map[string]string{
					"my-anno": "my-value",
				},
				Labels: map[string]string{
					"my-label": "my-value",
				},
			},
			Spec: spec.PluginSpec{
				Sidecar: spec.Sidecar{
					AutomountServiceAccountToken: true,
					Container: apiv1.Container{
						Ports: []apiv1.ContainerPort{{ContainerPort: 1234}},
						Resources: apiv1.ResourceRequirements{
							Limits:   map[apiv1.ResourceName]resource.Quantity{},
							Requests: map[apiv1.ResourceName]resource.Quantity{},
						},
						SecurityContext: &apiv1.SecurityContext{},
					},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "my-plug-executor-plugin", cm.Name)
		assert.Len(t, cm.Annotations, 1)
		assert.Equal(t, map[string]string{
			"my-label":                             "my-value",
			"workflows.argoproj.io/configmap-type": "ExecutorPlugin",
		}, cm.Labels)
		assert.Equal(t, map[string]string{
			"sidecar.automountServiceAccountToken": "true",
			"sidecar.container":                    "name: \"\"\nports:\n- containerPort: 1234\nresources: {}\nsecurityContext: {}\n",
		}, cm.Data)
	})
}

func TestFromConfigMap(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		_, err := FromConfigMap(&apiv1.ConfigMap{})
		require.EqualError(t, err, "sidecar is invalid: at least one port is mandatory")
	})
	t.Run("Valid", func(t *testing.T) {
		p, err := FromConfigMap(&apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-plug-executor-plugin",
				Annotations: map[string]string{
					"my-anno": "my-value",
				},
				Labels: map[string]string{
					common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapExecutorPlugin,
					"my-label":                   "my-value",
				},
			},
			Data: map[string]string{
				"sidecar.automountServiceAccountToken": "true",
				"sidecar.container":                    "{'name': 'my-name', 'ports': [{}], 'resources': {'requests': {}, 'limits': {}}, 'securityContext': {}}",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "ExecutorPlugin", p.Kind)
		assert.Equal(t, "my-plug", p.Name)
		assert.Len(t, p.Annotations, 1)
		assert.Len(t, p.Labels, 1)
		assert.True(t, p.Spec.Sidecar.AutomountServiceAccountToken)
		assert.Equal(t, apiv1.Container{
			Name:  "my-name",
			Ports: []apiv1.ContainerPort{{}},
			Resources: apiv1.ResourceRequirements{
				Limits:   map[apiv1.ResourceName]resource.Quantity{},
				Requests: map[apiv1.ResourceName]resource.Quantity{},
			},
			SecurityContext: &apiv1.SecurityContext{},
		}, p.Spec.Sidecar.Container)
	})
}
