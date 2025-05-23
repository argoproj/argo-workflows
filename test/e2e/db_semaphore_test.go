//go:build dbsemaphore

package e2e

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/test/e2e/fixtures"
)

type DBSemaphoreSuite struct {
	fixtures.E2ESuite
}

func (s *DBSemaphoreSuite) TestSynchronizationWfLevelMutex() {
	s.Given().
		Workflow("@synchronization/db-mutex-wf-level-1.yaml").
		When().
		ClearDBSemaphoreState().
		SubmitWorkflow().
		Given().
		Workflow("@synchronization/db-mutex-wf-level.yaml").
		When().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeWaitingOnAMutex, 90*time.Second).
		WaitForWorkflow(fixtures.ToBeSucceeded, 90*time.Second)
}

func (s *DBSemaphoreSuite) TestTemplateLevelMutex() {
	s.Given().
		Workflow("@synchronization/db-mutex-tmpl-level.yaml").
		When().
		ClearDBSemaphoreState().
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeWaitingOnAMutex, 90*time.Second).
		WaitForWorkflow(fixtures.ToBeSucceeded, 90*time.Second)
}

func (s *DBSemaphoreSuite) TestWorkflowLevelSemaphore() {
	s.Given().
		Workflow("@synchronization/db-semaphore-wf-level.yaml").
		When().
		ClearDBSemaphoreState().
		SetupDatabaseSemaphore("argo/workflow", 1).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToHavePhase(wfv1.WorkflowUnknown), 90*time.Second).
		WaitForWorkflow().
		Then().
		When().
		WaitForWorkflow(fixtures.ToBeSucceeded, 90*time.Second)
}

func (s *DBSemaphoreSuite) TestTemplateLevelSemaphore() {
	s.Given().
		Workflow("@synchronization/db-semaphore-tmpl-level.yaml").
		When().
		ClearDBSemaphoreState().
		SetupDatabaseSemaphore("argo/template", 1).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeRunning, 90*time.Second).
		Then().
		ExpectWorkflow(func(t *testing.T, metadata *metav1.ObjectMeta, status *wfv1.WorkflowStatus) {
			assert.True(t, status.Nodes.Any(func(n wfv1.NodeStatus) bool {
				return strings.Contains(n.Message, "Waiting for")
			}))
		}).
		When().
		WaitForWorkflow(time.Second * 90)
}

func (s *DBSemaphoreSuite) TestSynchronizationTmplLevelMutexAndSemaphore() {
	s.Given().
		Workflow("@synchronization/db-tmpl-level-mutex-semaphore.yaml").
		When().
		ClearDBSemaphoreState().
		SetupDatabaseSemaphore("argo/workflow", 1).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, 90*time.Second)
}

func (s *DBSemaphoreSuite) TestSynchronizationMultiple() {
	s.Given().
		Workflow("@synchronization/db-multiple.yaml").
		When().
		ClearDBSemaphoreState().
		SetupDatabaseSemaphore("argo/workflow", 2).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, 90*time.Second)
}

// Legacy CRD entries: mutex and semaphore
func (s *DBSemaphoreSuite) TestSynchronizationLegacyMutexAndSemaphore() {
	s.Given().
		Workflow("@synchronization/db-legacy-mutex-semaphore.yaml").
		When().
		ClearDBSemaphoreState().
		SetupDatabaseSemaphore("argo/workflow", 1).
		SubmitWorkflow().
		WaitForWorkflow(fixtures.ToBeSucceeded, 90*time.Second)
}

func (s *DBSemaphoreSuite) TestSynchronizationCases() {
	type test struct {
		semaphore      bool
		semaphoreLimit int
		setup          []whenFunc
		condition      fixtures.Condition
	}
	otherClusterName := "other"
	deadClusterTimeout := -20 * time.Minute
	tests := map[string]test{
		"MutexAlreadyTaken": {
			semaphore: false,
			setup: []whenFunc{
				setupState("argo/mutex-already-taken", "argo/some-workflow", nil, true, true, 0, 0),
			},
			condition: fixtures.ToBeWaitingOnAMutex,
		},
		"SemaphoreAlreadyTaken": {
			semaphore:      true,
			semaphoreLimit: 1,
			setup: []whenFunc{
				setupState("argo/semaphore-already-taken", "argo/some-workflow", nil, false, true, 0, 0),
			},
			condition: fixtures.ToBeWaitingOnASemaphore,
		},
		// Another live cluster has our mutex, we should wait
		"MutexOtherClusterPending": {
			semaphore: false,
			setup: []whenFunc{
				setupHeartbeat(otherClusterName, 0),
				setupState("argo/mutex-other-cluster-pending", "argo/some-workflow", &otherClusterName, true, false, 0, -1*time.Second),
			},
			condition: fixtures.ToBeWaitingOnAMutex,
		},
		// Another live cluster has our semaphore, we should wait
		"SemaphoreOtherClusterPending": {
			semaphore:      true,
			semaphoreLimit: 1,
			setup: []whenFunc{
				setupHeartbeat(otherClusterName, 0),
				setupState("argo/semaphore-other-cluster-pending", "argo/some-workflow", &otherClusterName, false, false, 0, -1*time.Second),
			},
			condition: fixtures.ToBeWaitingOnASemaphore,
		},
		// Another dead cluster has our mutex, we should run
		"MutexDeadClusterPending": {
			semaphore: false,
			setup: []whenFunc{
				setupHeartbeat(otherClusterName, deadClusterTimeout),
				setupState("argo/mutex-dead-cluster-pending", "argo/some-workflow", &otherClusterName, true, false, 0, 0),
			},
			condition: fixtures.ToBeSucceeded,
		},
		// Another dead cluster has our semaphore, we should run
		"SemaphoreDeadClusterPending": {
			semaphore:      true,
			semaphoreLimit: 1,
			setup: []whenFunc{
				setupHeartbeat(otherClusterName, deadClusterTimeout),
				setupState("argo/semaphore-dead-cluster-pending", "argo/some-workflow", &otherClusterName, false, false, 0, 0),
			},
			condition: fixtures.ToBeSucceeded,
		},
		"SemaphorePriorityOrderingHigh": {
			semaphore:      true,
			semaphoreLimit: 1,
			setup: []whenFunc{
				// other-workflow doesn't actually exist, but we're using it to test priority ordering
				setupState("argo/semaphore-priority-ordering-high", "other-workflow-prio-high", nil, false, false, 0, -5*time.Minute),
				setupState("argo/semaphore-priority-ordering-high", "argo/semaphore-priority-ordering-high", nil, false, false, 100, 0),
			},
			condition: fixtures.ToBeSucceeded,
		},
		"SemaphorePriorityOrderingLow": {
			semaphore:      true,
			semaphoreLimit: 1,
			setup: []whenFunc{
				// other-workflow doesn't actually exist, but we're using it to test priority ordering
				setupState("argo/semaphore-priority-ordering-low", "other-workflow-prio-low", nil, false, false, 100, -5*time.Minute),
				setupState("argo/semaphore-priority-ordering-low", "argo/semaphore-priority-ordering-low", nil, false, false, 0, 0),
			},
			condition: fixtures.ToBeWaitingOnASemaphore,
		},
	}

	// Iterate over tests in sorted order
	for testName, testCase := range tests {
		s.T().Run(testName, func(t *testing.T) {
			workflowName := testName
			for i := 0; i < len(workflowName); i++ {
				if i > 0 && workflowName[i] >= 'A' && workflowName[i] <= 'Z' {
					// Insert - before capital letters
					workflowName = workflowName[:i] + "-" + workflowName[i:]
					i++ // Skip the underscore we just added
				}
			}
			workflowName = strings.ToLower(workflowName)

			semaphores := make([]*wfv1.SemaphoreRef, 0)
			mutexes := make([]*wfv1.Mutex, 0)
			if testCase.semaphore {
				semaphores = append(semaphores, &wfv1.SemaphoreRef{
					Database: &wfv1.SyncDatabaseRef{
						Key: workflowName,
					},
				})
				testCase.setup = append(testCase.setup,
					setupSemaphore(fmt.Sprintf("argo/%s", workflowName), testCase.semaphoreLimit))
			} else {
				mutexes = append(mutexes, &wfv1.Mutex{
					Database: true,
					Name:     workflowName,
				})
			}
			wf := wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name: workflowName,
				},
				Spec: wfv1.WorkflowSpec{
					Synchronization: &wfv1.Synchronization{
						Semaphores: semaphores,
						Mutexes:    mutexes,
					},
					Entrypoint: "main",
					Templates: []wfv1.Template{{
						Name: "main",
						Container: &apiv1.Container{
							Image: "argoproj/argosay:v2",
						},
					}},
				},
			}

			when := s.Given().
				WorkflowWorkflow(&wf).
				When().
				ClearDBSemaphoreState()
			setupWith(when, testCase.setup...)
			when.SubmitWorkflow().
				WaitForWorkflow(testCase.condition, 30*time.Second).
				DeleteWorkflow()
		})
	}
}

// Define a type for functions that operate on a When object
type whenFunc func(*fixtures.When) *fixtures.When

func setupSemaphore(name string, limit int) whenFunc {
	return func(w *fixtures.When) *fixtures.When {
		return w.SetupDatabaseSemaphore(name, limit)
	}
}

// setupState returns a whenFunc that sets a semaphore state with relative timestamp
// As we can't correctly time.Now() at test setup time, we need to pass in a relative offset
func setupState(name string, workflowKey string, controller *string, mutex bool, held bool, priority int32, nowOffset time.Duration) whenFunc {
	return func(w *fixtures.When) *fixtures.When {
		return w.SetDBSemaphoreState(name, workflowKey, controller, mutex, held, priority, time.Now().Add(nowOffset))
	}
}

// setupHeartbeat returns a whenFunc that sets a semaphore heartbeat with relative timestamp
func setupHeartbeat(name string, nowOffset time.Duration) whenFunc {
	return func(w *fixtures.When) *fixtures.When {
		return w.SetDBSemaphoreControllerHB(&name, time.Now().Add(nowOffset))
	}
}

func setupWith(when *fixtures.When, whenFuncs ...whenFunc) *fixtures.When {
	// Apply all the when functions in sequence
	for _, f := range whenFuncs {
		when = f(when)
	}

	return when
}

func TestDBSemaphoreSuite(t *testing.T) {
	suite.Run(t, new(DBSemaphoreSuite))
}
