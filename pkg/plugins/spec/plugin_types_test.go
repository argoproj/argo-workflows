package spec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestPlugin_Validate(t *testing.T) {
	err := Plugin{}.Validate()
	assert.EqualError(t, err, "sidecar is invalid: at least one port is mandatory")
}

func TestSidecar_Validate(t *testing.T) {
	t.Run("NoPorts", func(t *testing.T) {
		assert.EqualError(t, Sidecar{}.Validate(), "at least one port is mandatory")
	})
	t.Run("ResourceRequestsMissing", func(t *testing.T) {
		assert.EqualError(t, Sidecar{
			Container: apiv1.Container{Ports: []apiv1.ContainerPort{{}}},
		}.Validate(), "resources requests are mandatory")
	})
	t.Run("ResourceLimitsMissing", func(t *testing.T) {
		assert.EqualError(t,
			Sidecar{
				Container: apiv1.Container{
					Ports: []apiv1.ContainerPort{{}},
					Resources: apiv1.ResourceRequirements{
						Requests: map[apiv1.ResourceName]resource.Quantity{apiv1.ResourceCPU: resource.MustParse("1")},
					},
				},
			}.Validate(), "resources limits are mandatory")
	})
	t.Run("SecurityContext", func(t *testing.T) {
		assert.EqualError(t, Sidecar{
			Container: apiv1.Container{
				Ports: []apiv1.ContainerPort{{}},
				Resources: apiv1.ResourceRequirements{
					Requests: map[apiv1.ResourceName]resource.Quantity{apiv1.ResourceCPU: resource.MustParse("1")},
					Limits:   map[apiv1.ResourceName]resource.Quantity{apiv1.ResourceCPU: resource.MustParse("1")},
				},
			},
		}.Validate(), "security context is mandatory")
	})
}
