//go:build executor
// +build executor

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type SecureSuite struct {
	fixtures.E2ESuite
}

func (s *SecureSuite) TestSecure() {
	s.Given().
		Workflow("@testdata/basic-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectPods(func(t *testing.T, pods []apiv1.Pod) {
			for _, pod := range pods {
				assert.Equal(t, "workflow", pod.Spec.ServiceAccountName)
				assert.False(t, *pod.Spec.AutomountServiceAccountToken)
				assert.True(t, *pod.Spec.SecurityContext.RunAsNonRoot)
				assert.Equal(t, int64(8737), *pod.Spec.SecurityContext.RunAsUser)
				assert.Equal(t, int64(8737), *pod.Spec.SecurityContext.RunAsGroup)

				for _, c := range pod.Spec.Containers {
					assert.Equal(t, apiv1.ResourceRequirements{
						Requests: map[apiv1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("10m"),
							"memory": resource.MustParse("64Mi"),
						},
						Limits: map[apiv1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("500m"),
							"memory": resource.MustParse("128Mi"),
						},
					}, c.Resources)
					assert.True(t, *c.SecurityContext.RunAsNonRoot)
					assert.Equal(t, int64(8737), *c.SecurityContext.RunAsUser)
					assert.Equal(t, int64(8737), *c.SecurityContext.RunAsUser)
					assert.Equal(t, int64(8737), *c.SecurityContext.RunAsGroup)
					assert.False(t, *c.SecurityContext.AllowPrivilegeEscalation)
					assert.Equal(t, &apiv1.Capabilities{
						Drop: []apiv1.Capability{"ALL"},
					}, c.SecurityContext.Capabilities)
				}
			}
		})
}

func TestSecureSuite(t *testing.T) {
	suite.Run(t, new(SecureSuite))
}
