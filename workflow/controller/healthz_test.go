package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestHealthz(t *testing.T) {

	veryOldUnreconciledWF := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	veryOldUnreconciledWF.SetCreationTimestamp(metav1.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)) // a long time ago
	veryOldUnreconciledWF.SetName(veryOldUnreconciledWF.Name + "-1")

	newUnreconciledWF := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	newUnreconciledWF.SetCreationTimestamp(metav1.Now())
	newUnreconciledWF.SetName(newUnreconciledWF.Name + "-2")

	veryOldReconciledWF := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	veryOldReconciledWF.SetCreationTimestamp(metav1.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)) // a long time ago
	veryOldReconciledWF.SetName(veryOldUnreconciledWF.Name + "-3")
	veryOldReconciledWF.Labels = map[string]string{common.LabelKeyPhase: string(wfv1.WorkflowPending)}

	tests := []struct {
		workflows      []*wfv1.Workflow
		expectedStatus int
	}{
		{
			[]*wfv1.Workflow{veryOldUnreconciledWF},
			500,
		},
		{
			[]*wfv1.Workflow{newUnreconciledWF},
			200,
		},
		{
			[]*wfv1.Workflow{veryOldUnreconciledWF, newUnreconciledWF},
			500,
		},
		{
			[]*wfv1.Workflow{veryOldReconciledWF},
			200,
		},
	}

	for _, tt := range tests {
		workflowsAsInterfaceSlice := []interface{}{}
		for _, wf := range tt.workflows {
			workflowsAsInterfaceSlice = append(workflowsAsInterfaceSlice, wf)
		}
		cancel, controller := newController(workflowsAsInterfaceSlice...)
		defer cancel()

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(controller.Healthz)

		req, err := http.NewRequest("GET", "/healthz", nil)
		if err != nil {
			t.Fatal(err)
		}
		handler.ServeHTTP(rr, req)

		// Check the status code is what we expect.
		if status := rr.Code; status != tt.expectedStatus {
			t.Errorf("handler returned wrong status code: got %v want %v",
				status, tt.expectedStatus)
		}
	}

}
