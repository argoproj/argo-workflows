//go:build functional
// +build functional

package e2e

import (
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/client-go/applyconfigurations/meta/v1"
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
        image: alpine
        command:
          - sh
          - -c
          - |
            echo "foo"
`).When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.Equal(t, status.Phase, v1alpha1.WorkflowSuceeded)
		}).
		ExpectWorkflowNode(func(status v1alpha1.nodeStatus) bool {
			return strings.Contains(status.name, ".split")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		}).
		ExpectWorkflowNode(func(status v1alpha1.nodeStatus) bool {
			return strings.Contains(status.name, ".map")
		}, func(t *testing.T, status *v1alpha1.NodeStatus, pod *apiv1.Pod) {
			assert.Equal(t, v1alpha1.NodeSucceeded, status.Phase)
		})
}

func TestExprLangSSuite(t *testing.T) {
	suite.Run(t, new(ExprSuite))
}
