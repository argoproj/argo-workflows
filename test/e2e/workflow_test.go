//go:build functional
// +build functional

package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type WorkflowSuite struct {
	fixtures.E2ESuite
}

func (s *WorkflowSuite) TestContainerTemplateAutomountServiceAccountTokenDisabled() {
	s.Given().Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: get-resources-via-container-template-
  namespace: argo
spec:
  serviceAccountName: argo
  automountServiceAccountToken: false
  executor:
    serviceAccountName: argo 
  entrypoint: main
  templates:
    - name: main
      container:
        name: main
        image: bitnami/kubectl
        command:
          - sh
        args:
          - -c
          - |
           kubectl get cm 
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute*11).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowSuite) TestScriptTemplateAutomountServiceAccountTokenDisabled() {
	s.Given().Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: get-resources-via-script-template-
  namespace: argo
spec:
  serviceAccountName: argo
  automountServiceAccountToken: false
  executor:
    serviceAccountName: argo
  entrypoint: main
  templates:
    - name: main
      script:
        name: main
        image: bitnami/kubectl
        command:
          - sh
        source:
          kubectl get cm 
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, time.Minute*11).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowSucceeded, status.Phase)
		})
}

func (s *WorkflowSuite) TestWorkflowFailedWhenAllPodSetFailedFromPending() {
	(s.Given().Workflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: active-deadline-fanout-template-level-
  namespace: argo
spec:
  entrypoint: entrypoint
  templates:
  - name: entrypoint
    steps:
    - - name: fanout
        template: echo
        arguments:
          parameters:
            - name: item
              value: "{{item}}"
        withItems:
          - 1
          - 2
          - 3
          - 4
  - name: echo
    inputs:
      parameters:
        - name: item
    container:
      image: centos:latest
      imagePullPolicy: Always
      command:
        - sh
        - '-c'
      args:
        - echo
        - 'workflow number {{inputs.parameters.item}}'
        - sleep
        - '20'
    activeDeadlineSeconds: 2 # defined on template level, not workflow level !
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed, time.Minute*11).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowFailed, status.Phase)
			for _, node := range status.Nodes {
				if node.Type == wfv1.NodeTypePod {
					assert.Equal(t, wfv1.NodeFailed, node.Phase)
					assert.Contains(t, node.Message, "Pod was active on the node longer than the specified deadline")
				}
			}
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "fanout(0:1)")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			for _, c := range pod.Status.ContainerStatuses {
				if c.Name == common.WaitContainerName && c.State.Terminated == nil {
					assert.NotNil(t, c.State.Waiting)
					assert.Contains(t, c.State.Waiting.Reason, "PodInitializing")
					assert.Nil(t, c.State.Running)
				}
			}
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "fanout(1:2)")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			for _, c := range pod.Status.ContainerStatuses {
				if c.Name == common.WaitContainerName && c.State.Terminated == nil {
					assert.NotNil(t, c.State.Waiting)
					assert.Contains(t, c.State.Waiting.Reason, "PodInitializing")
					assert.Nil(t, c.State.Running)
				}
			}
		})).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "fanout(2:3)")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			for _, c := range pod.Status.ContainerStatuses {
				if c.Name == common.WaitContainerName && c.State.Terminated == nil {
					assert.NotNil(t, c.State.Waiting)
					assert.Contains(t, c.State.Waiting.Reason, "PodInitializing")
					assert.Nil(t, c.State.Running)
				}
			}
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, "fanout(3:4)")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			for _, c := range pod.Status.ContainerStatuses {
				if c.Name == common.WaitContainerName && c.State.Terminated == nil {
					assert.NotNil(t, c.State.Waiting)
					assert.Contains(t, c.State.Waiting.Reason, "PodInitializing")
					assert.Nil(t, c.State.Running)
				}
			}
		})
}

func TestWorkflowSuite(t *testing.T) {
	suite.Run(t, new(WorkflowSuite))
}
