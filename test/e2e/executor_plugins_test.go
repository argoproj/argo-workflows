//go:build plugins

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
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
			require.Len(t, pods, 1)
			pod := pods[0]
			spec := pod.Spec
			assert.Equal(t, ptr.To(false), spec.AutomountServiceAccountToken)
			assert.Equal(t, &apiv1.PodSecurityContext{
				RunAsUser:      nil,
				RunAsNonRoot:   ptr.To(true),
				SeccompProfile: &v1.SeccompProfile{Type: "RuntimeDefault"},
			}, spec.SecurityContext)
			require.Len(t, spec.Volumes, 4)
			assert.Contains(t, spec.Volumes[0].Name, "kube-api-access-")
			assert.Equal(t, "var-run-argo", spec.Volumes[1].Name)
			assert.Contains(t, spec.Volumes[2].Name, "kube-api-access-")
			assert.Equal(t, "argo-workflows-agent-ca-certificates", spec.Volumes[3].Name)

			require.Len(t, spec.Containers, 2)
			{
				plug := spec.Containers[0]
				require.Equal(t, "hello-executor-plugin", plug.Name)
				require.Len(t, plug.VolumeMounts, 2)
				assert.Equal(t, "var-run-argo", plug.VolumeMounts[0].Name)
				assert.Contains(t, plug.VolumeMounts[1].Name, "kube-api-access-")
			}
			{
				agent := spec.Containers[1]
				require.Equal(t, "main", agent.Name)
				require.Len(t, agent.VolumeMounts, 3)
				assert.Equal(t, "var-run-argo", agent.VolumeMounts[0].Name)
				assert.Contains(t, agent.VolumeMounts[1].Name, "kube-api-access-")
				assert.Equal(t, "argo-workflows-agent-ca-certificates", agent.VolumeMounts[2].Name)
				assert.Equal(t, &apiv1.SecurityContext{
					RunAsUser:                nil,
					RunAsNonRoot:             ptr.To(true),
					AllowPrivilegeEscalation: ptr.To(false),
					ReadOnlyRootFilesystem:   ptr.To(true),
					Privileged:               ptr.To(false),
					Capabilities:             &apiv1.Capabilities{Drop: []apiv1.Capability{"ALL"}},
					SeccompProfile:           &v1.SeccompProfile{Type: "RuntimeDefault"},
				}, agent.SecurityContext)
			}
		}).
		ExpectWorkflowTaskSet(func(t *testing.T, wfts *wfv1.WorkflowTaskSet) {
			assert.NotNil(t, wfts)
			assert.Empty(t, wfts.Spec.Tasks)
			assert.Empty(t, wfts.Status.Nodes)
			assert.Equal(t, "true", wfts.Labels[common.LabelKeyCompleted])
		})
}

func TestExecutorPluginsSuite(t *testing.T) {
	suite.Run(t, new(ExecutorPluginsSuite))
}
