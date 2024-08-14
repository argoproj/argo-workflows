//go:build plugins

package e2e

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

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
			require.Contains(t, n.Message, "Hello")
			require.Len(t, n.Outputs.Parameters, 1)
		}).
		ExpectPods(func(t *testing.T, pods []apiv1.Pod) {
			if require.Len(t, pods, 1) {
				pod := pods[0]
				spec := pod.Spec
				require.Equal(t, pointer.Bool(false), spec.AutomountServiceAccountToken)
				require.Equal(t, &apiv1.PodSecurityContext{
					RunAsUser:      pointer.Int64(8737),
					RunAsNonRoot:   pointer.Bool(true),
					SeccompProfile: &v1.SeccompProfile{Type: "RuntimeDefault"},
				}, spec.SecurityContext)
				if require.Len(t, spec.Volumes, 4) {
					require.Contains(t, spec.Volumes[0].Name, "kube-api-access-")
					require.Equal(t, "var-run-argo", spec.Volumes[1].Name)
					require.Contains(t, spec.Volumes[2].Name, "kube-api-access-")
					require.Equal(t, "argo-workflows-agent-ca-certificates", spec.Volumes[3].Name)
				}
				if require.Len(t, spec.Containers, 2) {
					{
						plug := spec.Containers[0]
						if require.Equal(t, "hello-executor-plugin", plug.Name) {
							if require.Len(t, plug.VolumeMounts, 2) {
								require.Equal(t, "var-run-argo", plug.VolumeMounts[0].Name)
								require.Contains(t, plug.VolumeMounts[1].Name, "kube-api-access-")
							}
						}
					}
					{
						agent := spec.Containers[1]
						if require.Equal(t, "main", agent.Name) {
							if require.Len(t, agent.VolumeMounts, 3) {
								require.Equal(t, "var-run-argo", agent.VolumeMounts[0].Name)
								require.Contains(t, agent.VolumeMounts[1].Name, "kube-api-access-")
								require.Equal(t, "argo-workflows-agent-ca-certificates", agent.VolumeMounts[2].Name)
							}
							require.Equal(t, &apiv1.SecurityContext{
								RunAsUser:                pointer.Int64(8737),
								RunAsNonRoot:             pointer.Bool(true),
								AllowPrivilegeEscalation: pointer.Bool(false),
								ReadOnlyRootFilesystem:   pointer.Bool(true),
								Privileged:               pointer.Bool(false),
								Capabilities:             &apiv1.Capabilities{Drop: []apiv1.Capability{"ALL"}},
								SeccompProfile:           &v1.SeccompProfile{Type: "RuntimeDefault"},
							}, agent.SecurityContext)
						}
					}
				}
			}
		}).
		ExpectWorkflowTaskSet(func(t *testing.T, wfts *wfv1.WorkflowTaskSet) {
			require.NotNil(t, wfts)
			require.Empty(t, wfts.Spec.Tasks)
			require.Empty(t, wfts.Status.Nodes)
			require.Equal(t, "true", wfts.Labels[common.LabelKeyCompleted])
		})
}

func TestExecutorPluginsSuite(t *testing.T) {
	suite.Run(t, new(ExecutorPluginsSuite))
}
