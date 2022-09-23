//go:build functional
// +build functional

package e2e

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type RetryTestSuite struct {
	fixtures.E2ESuite
}

func (s *RetryTestSuite) TestRetryLimit() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-retry-limit-
spec:
  entrypoint: main
  templates:
    - name: main
      retryStrategy:
        limit: 0
        backoff:
          duration: 2s
          factor: 2
          maxDuration: 5m
      container:
        name: main
        image: 'argoproj/argosay:v2'
        args: [ exit, "1" ]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeFailed).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowPhase("Failed"), status.Phase)
			assert.Equal(t, "No more retries left", status.Message)
		})
}

func (s *RetryTestSuite) TestRetryBackoff() {
	s.Given().
		Workflow(`
metadata:
  generateName: test-backoff-strategy-
spec:
  entrypoint: main
  templates:
    - name: main
      retryStrategy:
        limit: '10'
        backoff:
          duration: 10s
          maxDuration: 1m
      container:
          name: main
          image: 'argoproj/argosay:v2'
          args: [ exit, "1" ]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowPhase("Failed"), status.Phase)
			assert.LessOrEqual(t, len(status.Nodes), 10)
		})
	s.Given().
		Workflow(`
metadata:
  generateName: test-backoff-strategy-
spec:
  entrypoint: main
  templates:
    - name: main
      retryStrategy:
        limit: 10
        backoff:
          duration: 10s
          maxDuration: 1m
      container:
          name: main
          image: 'argoproj/argosay:v2'
          args: [ exit, "1" ]
`).
		When().
		SubmitWorkflow().
		WaitForWorkflow(time.Minute).
		Then().
		ExpectWorkflow(func(t *testing.T, _ *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.Equal(t, wfv1.WorkflowPhase("Failed"), status.Phase)
			assert.LessOrEqual(t, len(status.Nodes), 10)
		})
}

func TestRetrySuite(t *testing.T) {
	suite.Run(t, new(RetryTestSuite))
}
