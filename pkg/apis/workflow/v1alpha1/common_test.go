package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareByPriority(t *testing.T) {

	stepOne := WorkflowStep{Name: "A", Priority: 100}
	stepTwo := WorkflowStep{Name: "B", Priority: 50}

	// true since priority of A is greater than B
	assert.True(t, CompareByPriority(&stepOne, &stepTwo))

	stepOne.Priority = 0
	stepTwo.Priority = 0

	assert.False(t, CompareByPriority(&stepOne, &stepTwo))

}
