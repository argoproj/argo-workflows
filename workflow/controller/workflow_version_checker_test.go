package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testWorkflow struct {
	metav1.ObjectMeta
}

func (t *testWorkflow) GetResourceVersion() string {
	return t.ResourceVersion
}

func TestWorkflowVersionChecker(t *testing.T) {
	checker := NewWorkflowVersionChecker(NoExpiration)

	// Test outdated version tracking
	wf1 := &testWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-workflow",
			Namespace:       "default",
			ResourceVersion: "1",
		},
	}

	// Mark version 1 as outdated
	checker.UpdateOutdatedVersion(wf1)

	// Verify version 1 is marked as outdated
	assert.True(t, checker.IsOutdated(wf1), "Version 1 should be marked as outdated")

	// Test different workflow
	wf2 := &testWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-workflow",
			Namespace:       "default",
			ResourceVersion: "2",
		},
	}
	assert.False(t, checker.IsOutdated(wf2), "Version 2 should not be marked as outdated")
}

func TestWorkflowVersionCheckerWithDifferentNamespaces(t *testing.T) {
	checker := NewWorkflowVersionChecker(10 * time.Minute)

	// Create workflows in different namespaces
	wf1 := &testWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-workflow",
			Namespace:       "namespace1",
			ResourceVersion: "1",
		},
	}
	wf2 := &testWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-workflow",
			Namespace:       "namespace2",
			ResourceVersion: "1",
		},
	}

	// Mark version 1 in namespace1 as outdated
	checker.UpdateOutdatedVersion(wf1)

	// Check that they don't interfere with each other
	assert.True(t, checker.IsOutdated(wf1), "Workflow in namespace1 should be marked as outdated")
	assert.False(t, checker.IsOutdated(wf2), "Workflow in namespace2 should not be marked as outdated")
}

func TestWorkflowVersionCheckerWithDifferentNames(t *testing.T) {
	checker := NewWorkflowVersionChecker(10 * time.Minute)

	// Create workflows with different names
	wf1 := &testWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "workflow1",
			Namespace:       "default",
			ResourceVersion: "1",
		},
	}
	wf2 := &testWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "workflow2",
			Namespace:       "default",
			ResourceVersion: "1",
		},
	}

	// Mark workflow1 version 1 as outdated
	checker.UpdateOutdatedVersion(wf1)

	// Check that they don't interfere with each other
	assert.True(t, checker.IsOutdated(wf1), "workflow1 should be marked as outdated")
	assert.False(t, checker.IsOutdated(wf2), "workflow2 should not be marked as outdated")
}

func TestWorkflowVersionCheckerTTL(t *testing.T) {
	checker := NewWorkflowVersionChecker(time.Millisecond)

	// Create and mark a workflow version as outdated
	wf := &testWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-workflow",
			Namespace:       "default",
			ResourceVersion: "1",
		},
	}
	checker.UpdateOutdatedVersion(wf)

	// Wait for TTL to expire
	time.Sleep(time.Millisecond * 2)

	// Verify it's no longer marked as outdated
	assert.False(t, checker.IsOutdated(wf), "Version should no longer be marked as outdated after TTL")
}
