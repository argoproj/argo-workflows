package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type testWorkflow struct {
	metav1.ObjectMeta
}

func (t *testWorkflow) GetResourceVersion() string {
	return t.ResourceVersion
}

func TestWorkflowVersionChecker(t *testing.T) {
	tests := []struct {
		name             string
		ttl              time.Duration
		workflows        []*testWorkflow
		checkWorkflow    *testWorkflow
		expectedOutdated bool
	}{
		{
			name: "workflow is not outdated when no previous version exists",
			ttl:  10 * time.Minute,
			workflows: []*testWorkflow{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-workflow",
						Namespace:       "default",
						ResourceVersion: "1",
					},
				},
			},
			checkWorkflow: &testWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-workflow",
					Namespace:       "default",
					ResourceVersion: "1",
				},
			},
			expectedOutdated: false,
		},
		{
			name: "workflow is not outdated when previous version is the same as the current version",
			ttl:  10 * time.Minute,
			workflows: []*testWorkflow{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-workflow",
						Namespace:       "default",
						ResourceVersion: "1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-workflow",
						Namespace:       "default",
						ResourceVersion: "1",
					},
				},
			},
			checkWorkflow: &testWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-workflow",
					Namespace:       "default",
					ResourceVersion: "1",
				},
			},
			expectedOutdated: false,
		},
		{
			name: "workflow is outdated when previous version exists",
			ttl:  10 * time.Minute,
			workflows: []*testWorkflow{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-workflow",
						Namespace:       "default",
						ResourceVersion: "1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-workflow",
						Namespace:       "default",
						ResourceVersion: "2",
					},
				},
			},
			checkWorkflow: &testWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-workflow",
					Namespace:       "default",
					ResourceVersion: "1",
				},
			},
			expectedOutdated: true,
		},
		{
			name: "workflow is not outdated when it's the latest version",
			ttl:  10 * time.Minute,
			workflows: []*testWorkflow{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-workflow",
						Namespace:       "default",
						ResourceVersion: "1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-workflow",
						Namespace:       "default",
						ResourceVersion: "2",
					},
				},
			},
			checkWorkflow: &testWorkflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-workflow",
					Namespace:       "default",
					ResourceVersion: "2",
				},
			},
			expectedOutdated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewWorkflowVersionChecker(tt.ttl)

			// Update versions in order
			for _, wf := range tt.workflows {
				checker.UpdateLatestVersion(wf)
			}

			// Check if the workflow is outdated
			assert.Equal(t, tt.expectedOutdated, checker.IsOutdated(tt.checkWorkflow))
		})
	}
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

	// Update both workflows
	checker.UpdateLatestVersion(wf1)
	checker.UpdateLatestVersion(wf2)

	// Check that they don't interfere with each other
	assert.False(t, checker.IsOutdated(wf1), "Workflow in namespace1 should not be outdated")
	assert.False(t, checker.IsOutdated(wf2), "Workflow in namespace2 should not be outdated")
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

	// Update both workflows
	checker.UpdateLatestVersion(wf1)
	checker.UpdateLatestVersion(wf2)

	// Check that they don't interfere with each other
	assert.False(t, checker.IsOutdated(wf1), "workflow1 should not be outdated")
	assert.False(t, checker.IsOutdated(wf2), "workflow2 should not be outdated")
}

func TestWorkflowVersionCheckerNoTTL(t *testing.T) {
	checker := NewWorkflowVersionChecker(NoExpiration)

	// Create and update a workflow
	wf1 := &testWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-workflow",
			Namespace:       "default",
			ResourceVersion: "1",
		},
	}
	wf2 := &testWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-workflow",
			Namespace:       "default",
			ResourceVersion: "2",
		},
	}

	// Update with version 1
	checker.UpdateLatestVersion(wf1)

	// Version 1 should be outdated after version 2
	checker.UpdateLatestVersion(wf2)
	assert.True(t, checker.IsOutdated(wf1), "Version 1 should be outdated after version 2")
}

func TestWorkflowVersionChecker_MarkForExpiration(t *testing.T) {
	checker := NewWorkflowVersionChecker(time.Nanosecond)

	// Test successful case
	wf := &testWorkflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "test-wf",
			Namespace:       "default",
			ResourceVersion: "1",
		},
	}

	// First set a version
	checker.UpdateLatestVersion(wf)
	key, _ := cache.MetaNamespaceKeyFunc(wf)
	time.Sleep(time.Nanosecond)

	_, exists := checker.cache.Get(key)
	assert.True(t, exists)

	// Then mark for expiration
	checker.MarkForExpiration(wf)
	time.Sleep(time.Nanosecond)
	_, exists = checker.cache.Get(key)
	assert.True(t, exists)
}
