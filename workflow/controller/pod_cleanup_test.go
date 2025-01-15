package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo-workflows/v3/workflow/common"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func Test_determinePodCleanupAction(t *testing.T) {
	finalizersNotOurs := []string{}
	finalizersOurs := append(finalizersNotOurs, common.FinalizerPodStatus)
	assert.Equal(t, labelPodCompleted, determinePodCleanupAction(labels.Nothing(), nil, wfv1.PodGCOnPodCompletion, wfv1.WorkflowSucceeded, apiv1.PodSucceeded, finalizersOurs))
	assert.Equal(t, labelPodCompleted, determinePodCleanupAction(labels.Everything(), nil, wfv1.PodGCOnPodNone, wfv1.WorkflowSucceeded, apiv1.PodSucceeded, finalizersOurs))

	type fields = struct {
		Strategy      wfv1.PodGCStrategy `json:"strategy,omitempty"`
		WorkflowPhase wfv1.WorkflowPhase `json:"workflowPhase,omitempty"`
		PodPhase      apiv1.PodPhase     `json:"podPhase,omitempty"`
		Finalizers    []string
	}
	for _, tt := range []struct {
		Fields fields           `json:"fields"`
		Want   podCleanupAction `json:"want,omitempty"`
	}{

		// strategy = 4 options
		// workflow phase = 3 options
		// pod phase = 2 options

		// 4 * 3 * 2 = 24 options

		{fields{wfv1.PodGCOnPodNone, wfv1.WorkflowRunning, apiv1.PodSucceeded, finalizersNotOurs}, labelPodCompleted},
		{fields{wfv1.PodGCOnPodNone, wfv1.WorkflowRunning, apiv1.PodFailed, finalizersNotOurs}, labelPodCompleted},
		{fields{wfv1.PodGCOnPodNone, wfv1.WorkflowRunning, apiv1.PodSucceeded, finalizersOurs}, labelPodCompleted},
		{fields{wfv1.PodGCOnPodNone, wfv1.WorkflowRunning, apiv1.PodFailed, finalizersOurs}, labelPodCompleted},

		{fields{wfv1.PodGCOnWorkflowSuccess, wfv1.WorkflowRunning, apiv1.PodSucceeded, finalizersNotOurs}, ""},
		{fields{wfv1.PodGCOnWorkflowSuccess, wfv1.WorkflowRunning, apiv1.PodFailed, finalizersNotOurs}, ""},
		{fields{wfv1.PodGCOnWorkflowSuccess, wfv1.WorkflowRunning, apiv1.PodSucceeded, finalizersOurs}, removeFinalizer},
		{fields{wfv1.PodGCOnWorkflowSuccess, wfv1.WorkflowRunning, apiv1.PodFailed, finalizersOurs}, removeFinalizer},
		{fields{wfv1.PodGCOnWorkflowSuccess, wfv1.WorkflowSucceeded, apiv1.PodSucceeded, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnWorkflowSuccess, wfv1.WorkflowSucceeded, apiv1.PodFailed, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnWorkflowSuccess, wfv1.WorkflowFailed, apiv1.PodSucceeded, finalizersOurs}, labelPodCompleted},
		{fields{wfv1.PodGCOnWorkflowSuccess, wfv1.WorkflowFailed, apiv1.PodFailed, finalizersOurs}, labelPodCompleted},

		{fields{wfv1.PodGCOnWorkflowCompletion, wfv1.WorkflowRunning, apiv1.PodSucceeded, finalizersNotOurs}, ""},
		{fields{wfv1.PodGCOnWorkflowCompletion, wfv1.WorkflowRunning, apiv1.PodFailed, finalizersNotOurs}, ""},
		{fields{wfv1.PodGCOnWorkflowCompletion, wfv1.WorkflowRunning, apiv1.PodSucceeded, finalizersOurs}, removeFinalizer},
		{fields{wfv1.PodGCOnWorkflowCompletion, wfv1.WorkflowRunning, apiv1.PodFailed, finalizersOurs}, removeFinalizer},
		{fields{wfv1.PodGCOnWorkflowCompletion, wfv1.WorkflowSucceeded, apiv1.PodSucceeded, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnWorkflowCompletion, wfv1.WorkflowSucceeded, apiv1.PodFailed, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnWorkflowCompletion, wfv1.WorkflowFailed, apiv1.PodSucceeded, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnWorkflowCompletion, wfv1.WorkflowFailed, apiv1.PodFailed, finalizersOurs}, deletePod},

		{fields{wfv1.PodGCOnPodSuccess, wfv1.WorkflowRunning, apiv1.PodSucceeded, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnPodSuccess, wfv1.WorkflowRunning, apiv1.PodFailed, finalizersOurs}, labelPodCompleted},
		{fields{wfv1.PodGCOnPodSuccess, wfv1.WorkflowSucceeded, apiv1.PodSucceeded, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnPodSuccess, wfv1.WorkflowSucceeded, apiv1.PodFailed, finalizersOurs}, labelPodCompleted},
		{fields{wfv1.PodGCOnPodSuccess, wfv1.WorkflowFailed, apiv1.PodSucceeded, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnPodSuccess, wfv1.WorkflowFailed, apiv1.PodFailed, finalizersOurs}, labelPodCompleted},

		{fields{wfv1.PodGCOnPodCompletion, wfv1.WorkflowRunning, apiv1.PodSucceeded, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnPodCompletion, wfv1.WorkflowRunning, apiv1.PodFailed, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnPodCompletion, wfv1.WorkflowSucceeded, apiv1.PodSucceeded, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnPodCompletion, wfv1.WorkflowSucceeded, apiv1.PodFailed, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnPodCompletion, wfv1.WorkflowFailed, apiv1.PodSucceeded, finalizersOurs}, deletePod},
		{fields{wfv1.PodGCOnPodCompletion, wfv1.WorkflowFailed, apiv1.PodFailed, finalizersOurs}, deletePod},
	} {
		t.Run(wfv1.MustMarshallJSON(tt), func(t *testing.T) {
			action := determinePodCleanupAction(
				labels.Everything(),
				nil,
				tt.Fields.Strategy,
				tt.Fields.WorkflowPhase,
				tt.Fields.PodPhase,
				tt.Fields.Finalizers,
			)
			assert.Equal(t, tt.Want, action)
		})
	}
}
