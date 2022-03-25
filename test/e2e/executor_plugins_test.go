//go:build plugins
// +build plugins

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ExecutorPluginsSuite struct {
	fixtures.E2ESuite
}

func (s *ExecutorPluginsSuite) TestTemplateExecutor() {
	s.Given().
		Workflow("@testdata/plugins/executor/template-executor-workflow.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, md *metav1.ObjectMeta, s *wfv1.WorkflowStatus) {
			n := s.Nodes[md.Name]
			assert.Contains(t, n.Message, "Hello")
			assert.Len(t, n.Outputs.Parameters, 1)
		}).
		ExpectPods(func(t *testing.T, pods []apiv1.Pod) {
			if assert.Len(t, pods, 1) {
				pod := pods[0]
				spec := pod.Spec
				assert.Equal(t, pointer.BoolPtr(false), spec.AutomountServiceAccountToken)
				assert.Equal(t, &apiv1.PodSecurityContext{
					RunAsUser:    pointer.Int64(8737),
					RunAsNonRoot: pointer.BoolPtr(true),
				}, spec.SecurityContext)
				if assert.Len(t, spec.Volumes, 4) {
					assert.Contains(t, spec.Volumes[0].Name, "kube-api-access-")
					assert.Equal(t, spec.Volumes[1].Name, "var-run-argo")
					assert.Contains(t, spec.Volumes[2].Name, "kube-api-access-")
					assert.Equal(t, spec.Volumes[3].Name, "argo-workflows-agent-ca-certificates")
				}
				if assert.Len(t, spec.Containers, 2) {
					{
						plug := spec.Containers[0]
						if assert.Equal(t, "hello-executor-plugin", plug.Name) {
							if assert.Len(t, plug.VolumeMounts, 2) {
								assert.Equal(t, "var-run-argo", plug.VolumeMounts[0].Name)
								assert.Contains(t, plug.VolumeMounts[1].Name, "kube-api-access-")
							}
						}
					}
					{
						agent := spec.Containers[1]
						if assert.Equal(t, "main", agent.Name) {
							if assert.Len(t, agent.VolumeMounts, 3) {
								assert.Equal(t, "var-run-argo", agent.VolumeMounts[0].Name)
								assert.Contains(t, agent.VolumeMounts[1].Name, "kube-api-access-")
								assert.Equal(t, agent.VolumeMounts[2].Name, "argo-workflows-agent-ca-certificates")
							}
							assert.Equal(t, &apiv1.SecurityContext{
								RunAsUser:                pointer.Int64(8737),
								RunAsNonRoot:             pointer.BoolPtr(true),
								AllowPrivilegeEscalation: pointer.BoolPtr(false),
								ReadOnlyRootFilesystem:   pointer.BoolPtr(true),
								Capabilities:             &apiv1.Capabilities{Drop: []apiv1.Capability{"ALL"}},
							}, agent.SecurityContext)
						}
					}
				}
			}
		})
}

func TestExecutorPluginsSuite(t *testing.T) {
	suite.Run(t, new(ExecutorPluginsSuite))
}
