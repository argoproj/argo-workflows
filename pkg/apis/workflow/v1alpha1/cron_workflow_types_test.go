package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCronWorkflowConditions(t *testing.T) {
	cwfCond := CronWorkflowConditions{}
	cond := CronWorkflowCondition{
		Type:    CronWorkflowConditionSubmissionError,
		Message: "Failed to submit Workflow",
		Status:  v1.ConditionTrue,
	}

	assert.Len(t, cwfCond, 0)
	cwfCond.UpsertCondition(cond)
	assert.Len(t, cwfCond, 1)
	cwfCond.RemoveCondition(CronWorkflowConditionSubmissionError)
	assert.Len(t, cwfCond, 0)
}
