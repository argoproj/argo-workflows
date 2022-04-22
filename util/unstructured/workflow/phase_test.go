package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func TestGetPhase(t *testing.T) {
	un, _ := util.ToUnstructured(&wfv1.Workflow{})
	assert.Equal(t, GetPhase(un), wfv1.WorkflowUnknown)
	un, _ = util.ToUnstructured(&wfv1.Workflow{
		Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning},
	})
	assert.Equal(t, GetPhase(un), wfv1.WorkflowRunning)
}
