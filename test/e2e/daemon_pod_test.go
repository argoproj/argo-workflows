// +build functional

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type DaemonPodSuite struct {
	fixtures.E2ESuite
}

func (s *DaemonPodSuite) TestWorkflowCompletesIfContainsDaemonPod() {
	s.Given().
		Workflow(`
metadata:
  generateName: whalesay-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    dag:
      tasks:
        - name: redis
          template: redis-tmpl
        - name: whale
          dependencies: [redis]
          template: whale-tmpl
  - name: redis-tmpl
    daemon: true
    container:
      image: argoproj/argosay:v2
      args: ["sleep", "100s"]
  - name: whale-tmpl
    container:
      image: argoproj/argosay:v2
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *v1.ObjectMeta, status *v1alpha1.WorkflowStatus) {
			assert.False(t, status.FinishedAt.IsZero())
		})
}

func TestDaemonPodSuite(t *testing.T) {
	suite.Run(t, new(DaemonPodSuite))
}
