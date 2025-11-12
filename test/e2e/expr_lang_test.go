//go:build functional

package e2e

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type ExprSuite struct {
	fixtures.E2ESuite
}

func (s *ExprSuite) TestRegression12037() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: broken-
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: split
            template: foo
          - name: map
            template: foo
            depends: split

    - name: foo
      container:
        image: argoproj/argosay:v2
        command:
        - echo
        args:
        - foo
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, ".split")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.NodeStatus) bool {
			return strings.Contains(status.Name, ".map")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func (s *ExprSuite) TestRegression15008() {
	s.Given().
		Workflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: item-used-in-sprig-
spec:
  entrypoint: loop-example
  templates:
  - name: print-message
    container:
      image: argoproj/argosay:v2
      args:
      - '{{inputs.parameters.message}}'
      command:
      - echo
    inputs:
      parameters:
      - name: message
  - name: loop-example
    steps:
    - - name: print-message-loop
        template: print-message
        withItems:
        - example
        arguments:
          parameters:
          - name: message
            value: '{{=sprig.upper(item)}}'
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, v1alpha1.WorkflowSucceeded, status.Phase)

			stepNode := status.Nodes.FindByDisplayName("print-message-loop(0:example)")
			require.NotNil(t, stepNode)
			require.NotNil(t, stepNode.Inputs.Parameters[0].Value)
			assert.Equal(t, "EXAMPLE", string(*stepNode.Inputs.Parameters[0].Value))
		})
}

func TestExprLangSuite(t *testing.T) {
	suite.Run(t, new(ExprSuite))
}
