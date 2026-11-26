package controller

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/pod"
)

const (
	podCompletionNodeID = "workflow-1234"
	// Duty-cycle inputs seeded by newPodCompletionController.
	podReconcileCost    = time.Second
	podReconcileElapsed = time.Second
	podDutyCyclePercent = 20
	// A 1s reconcile at 20% needs a 4s idle gap; 1s of it has elapsed, so 3s remain.
	podCompletionDelay = 3 * time.Second
	podRoutineDelay    = 10 * time.Second
)

var podCompletionOwners = []metav1.OwnerReference{{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowKind, Name: "workflow", UID: "uid"}}

// newPodCompletionController builds a completion-delay controller with real
// TaskResult and pod informer indexers (never started) and seeded duty-cycle timing.
func newPodCompletionController(t *testing.T, floor *time.Duration) (*WorkflowController, *telemetry.TestMetricsExporter) {
	t.Helper()
	controller, exporter := newCompletionDelayController(t, podRoutineDelay, floor)
	controller.completionDutyCyclePercent = podDutyCyclePercent
	controller.reconcileTimings = map[types.UID]workflowReconcileTiming{
		"uid": {duration: podReconcileCost, lastEnd: time.Now().Add(-podReconcileElapsed)},
	}
	controller.taskResultInformer = cache.NewSharedIndexInformer(&cache.ListWatch{}, &wfv1.WorkflowTaskResult{}, 0, cache.Indexers{})
	controller.PodController = pod.TestingNewController(cache.NewSharedIndexInformer(&cache.ListWatch{}, &apiv1.Pod{}, 0,
		cache.Indexers{indexes.NodeIDIndex: indexes.MetaNodeIDIndexFunc}))
	return controller, exporter
}

func podCompletionPod(phase apiv1.PodPhase) *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "workflow-pod",
			Namespace:       "default",
			Labels:          map[string]string{common.LabelKeyWorkflow: "workflow"},
			Annotations:     map[string]string{common.AnnotationKeyNodeID: podCompletionNodeID},
			OwnerReferences: podCompletionOwners,
		},
		Status: apiv1.PodStatus{Phase: phase},
	}
}

// podCompletionTaskResult builds the node's TaskResult; an empty completed
// leaves the report-outputs-completed label off.
func podCompletionTaskResult(completed string) *wfv1.WorkflowTaskResult {
	result := &wfv1.WorkflowTaskResult{ObjectMeta: metav1.ObjectMeta{
		Name:            podCompletionNodeID,
		Namespace:       "default",
		Labels:          map[string]string{common.LabelKeyWorkflow: "workflow"},
		OwnerReferences: podCompletionOwners,
	}}
	if completed != "" {
		result.Labels[common.LabelKeyReportOutputsCompleted] = completed
	}
	return result
}

// assertEnqueuedAfter checks the workflow becomes ready exactly after want, and
// that the completion-delay metric was recorded only for completions.
func assertEnqueuedAfter(t *testing.T, controller *WorkflowController, exporter *telemetry.TestMetricsExporter, want time.Duration, completion bool) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	attrs := attribute.NewSet()
	data, err := exporter.GetFloat64HistogramData(ctx, telemetry.InstrumentWorkflowCompletionRequeueDelay.Name(), &attrs)
	if completion {
		require.NoError(t, err)
		assert.Equal(t, uint64(1), data.Count)
		assert.InDelta(t, want.Seconds(), data.Sum, 0.000001)
	} else {
		require.Error(t, err, "routine enqueues do not record a completion delay")
	}
	synctest.Wait()
	time.Sleep(want - time.Nanosecond)
	synctest.Wait()
	require.Zero(t, controller.wfQueue.Len())
	time.Sleep(time.Nanosecond)
	synctest.Wait()
	require.Equal(t, 1, controller.wfQueue.Len())
}

// TestPodCompletionDelay checks that a pod reaching Failed takes the completion
// delay at once, and a pod reaching Succeeded only when its TaskResult is cached
// with its final outputs, matching when the operator completes the node.
func TestPodCompletionDelay(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		disabled             bool
		oldPhase             *apiv1.PodPhase // nil is an add or delete event
		newPhase             apiv1.PodPhase
		taskResult           *wfv1.WorkflowTaskResult
		noNodeID             bool
		noTaskResultInformer bool
		completion           bool
	}{
		{name: "succeeded, result complete", oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodSucceeded, taskResult: podCompletionTaskResult("true"), completion: true},
		{name: "succeeded from pending, result complete", oldPhase: new(apiv1.PodPending), newPhase: apiv1.PodSucceeded, taskResult: podCompletionTaskResult("true"), completion: true},
		{name: "succeeded, result incomplete", oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodSucceeded, taskResult: podCompletionTaskResult("false")},
		{name: "succeeded, result not cached yet", oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodSucceeded},
		{name: "succeeded, missing node ID", oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodSucceeded, taskResult: podCompletionTaskResult("true"), noNodeID: true},
		{name: "succeeded, no TaskResult informer", oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodSucceeded, noTaskResultInformer: true},
		{name: "failed, result complete", oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodFailed, taskResult: podCompletionTaskResult("true"), completion: true},
		{name: "failed, result incomplete", oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodFailed, taskResult: podCompletionTaskResult("false"), completion: true},
		{name: "failed, result not cached", oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodFailed, completion: true},
		{name: "failed, missing node ID", oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodFailed, noNodeID: true, completion: true},
		{name: "already succeeded", oldPhase: new(apiv1.PodSucceeded), newPhase: apiv1.PodSucceeded, taskResult: podCompletionTaskResult("true")},
		{name: "already failed", oldPhase: new(apiv1.PodFailed), newPhase: apiv1.PodFailed},
		{name: "not finished", oldPhase: new(apiv1.PodPending), newPhase: apiv1.PodRunning},
		{name: "add or delete of finished pod", newPhase: apiv1.PodSucceeded, taskResult: podCompletionTaskResult("true")},
		{name: "add or delete of failed pod", newPhase: apiv1.PodFailed},
		{name: "delete of unfinished pod", newPhase: apiv1.PodRunning, taskResult: podCompletionTaskResult("true")},
		{name: "disabled, succeeded", disabled: true, oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodSucceeded, taskResult: podCompletionTaskResult("true")},
		{name: "disabled, failed", disabled: true, oldPhase: new(apiv1.PodRunning), newPhase: apiv1.PodFailed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				floor := new(time.Millisecond)
				if tt.disabled {
					floor = nil
				}
				controller, exporter := newPodCompletionController(t, floor)
				if tt.taskResult != nil {
					require.NoError(t, controller.taskResultInformer.GetIndexer().Add(tt.taskResult))
				}
				if tt.noTaskResultInformer {
					controller.taskResultInformer = nil
				}
				var oldPod *apiv1.Pod
				if tt.oldPhase != nil {
					oldPod = podCompletionPod(*tt.oldPhase)
				}
				newPod := podCompletionPod(tt.newPhase)
				if tt.noNodeID {
					newPod.Annotations = nil
				}
				require.NoError(t, controller.enqueueWfFromPodLabel(logging.TestContext(t.Context()), oldPod, newPod))
				want := podRoutineDelay
				if tt.completion {
					want = podCompletionDelay
				}
				assertEnqueuedAfter(t, controller, exporter, want, tt.completion)
			})
		})
	}
}

// TestEnqueueWfFromPodLabelRejectsUnrelatedPods checks that pods without the
// workflow label return an error and enqueue nothing.
func TestEnqueueWfFromPodLabelRejectsUnrelatedPods(t *testing.T) {
	for name, labels := range map[string]map[string]string{
		"no labels":         nil,
		"no workflow label": {"other": "label"},
	} {
		t.Run(name, func(t *testing.T) {
			controller, _ := newPodCompletionController(t, new(time.Millisecond))
			pod := podCompletionPod(apiv1.PodSucceeded)
			pod.Labels = labels
			require.Error(t, controller.enqueueWfFromPodLabel(logging.TestContext(t.Context()), podCompletionPod(apiv1.PodRunning), pod))
			assert.Zero(t, controller.wfQueue.Len())
		})
	}
}

// TestEnqueueNodeCompletionDisabled checks that with COMPLETED_REQUEUE_TIME_MS
// unset a node completion keeps the normal delay and records no metric.
func TestEnqueueNodeCompletionDisabled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		controller, exporter := newPodCompletionController(t, nil)
		controller.enqueueNodeCompletion(logging.TestContext(t.Context()), "default/workflow", podCompletionOwners)
		assertEnqueuedAfter(t, controller, exporter, podRoutineDelay, false)
	})
}

// TestTaskResultCompletionDelay drives the handlers the TaskResult informer
// registers, so it also checks they pass the old and new results in order. A
// TaskResult getting its final outputs takes the completion delay only when its pod,
// found by node ID, has already finished.
func TestTaskResultCompletionDelay(t *testing.T) {
	for _, tt := range []struct {
		name            string
		disabled        bool
		add             bool
		oldLabel        string
		newLabel        string
		podPhase        *apiv1.PodPhase // nil means no pod in the informer
		noPodController bool
		noNodeIDIndex   bool
		completion      bool
	}{
		{name: "pod succeeded", oldLabel: "false", newLabel: "true", podPhase: new(apiv1.PodSucceeded), completion: true},
		{name: "pod failed", oldLabel: "false", newLabel: "true", podPhase: new(apiv1.PodFailed), completion: true},
		{name: "pod running", oldLabel: "false", newLabel: "true", podPhase: new(apiv1.PodRunning)},
		{name: "pod missing", oldLabel: "false", newLabel: "true"},
		{name: "no pod controller", oldLabel: "false", newLabel: "true", noPodController: true},
		{name: "pod lookup fails", oldLabel: "false", newLabel: "true", noNodeIDIndex: true},
		{name: "label added as true", oldLabel: "", newLabel: "true", podPhase: new(apiv1.PodSucceeded), completion: true},
		{name: "label still absent", oldLabel: "", newLabel: "", podPhase: new(apiv1.PodSucceeded)},
		{name: "already complete", oldLabel: "true", newLabel: "true", podPhase: new(apiv1.PodSucceeded)},
		{name: "not complete", oldLabel: "false", newLabel: "false", podPhase: new(apiv1.PodSucceeded)},
		{name: "reverted to incomplete", oldLabel: "true", newLabel: "false", podPhase: new(apiv1.PodSucceeded)},
		{name: "add, complete", add: true, newLabel: "true", podPhase: new(apiv1.PodSucceeded)},
		{name: "disabled", disabled: true, oldLabel: "false", newLabel: "true", podPhase: new(apiv1.PodSucceeded)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				floor := new(time.Millisecond)
				if tt.disabled {
					floor = nil
				}
				controller, exporter := newPodCompletionController(t, floor)
				if tt.podPhase != nil {
					require.NoError(t, controller.PodController.TestingPodInformer().GetIndexer().Add(podCompletionPod(*tt.podPhase)))
				}
				if tt.noPodController {
					controller.PodController = nil
				}
				if tt.noNodeIDIndex {
					controller.PodController = pod.TestingNewController(cache.NewSharedIndexInformer(&cache.ListWatch{}, &apiv1.Pod{}, 0, cache.Indexers{}))
				}
				handlers := controller.taskResultEventHandlers(logging.TestContext(t.Context()))
				if tt.add {
					handlers.OnAdd(podCompletionTaskResult(tt.newLabel), false)
				} else {
					handlers.OnUpdate(podCompletionTaskResult(tt.oldLabel), podCompletionTaskResult(tt.newLabel))
				}
				want := podRoutineDelay
				if tt.completion {
					want = podCompletionDelay
				}
				assertEnqueuedAfter(t, controller, exporter, want, tt.completion)
			})
		})
	}
}

// TestPodCompletionLookupIndexes guards the cache keys both lookups rely on: the
// TaskResult store key is namespace/node ID and pods are indexed by node ID.
func TestPodCompletionLookupIndexes(t *testing.T) {
	key, err := cache.MetaNamespaceKeyFunc(podCompletionTaskResult("true"))
	require.NoError(t, err)
	assert.Equal(t, "default/"+podCompletionNodeID, key)
	values, err := indexes.MetaNodeIDIndexFunc(podCompletionPod(apiv1.PodSucceeded))
	require.NoError(t, err)
	assert.Equal(t, []string{"default/" + podCompletionNodeID}, values)
}
