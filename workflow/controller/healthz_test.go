package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHealthz(t *testing.T) {

	veryOldWF := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	veryOldWF.SetCreationTimestamp(metav1.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)) // a long time ago
	veryOldWF.SetName(veryOldWF.Name + "-1")

	newWF := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	newWF.SetCreationTimestamp(metav1.Now())
	newWF.SetName(newWF.Name + "-2")

	veryOldButReconciledWF := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	veryOldButReconciledWF.SetCreationTimestamp(metav1.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)) // a long time ago
	veryOldButReconciledWF.SetName(veryOldWF.Name + "-3")
	veryOldButReconciledWF.Labels = map[string]string{common.LabelKeyPhase: string(wfv1.WorkflowPending)}

	tests := []struct {
		workflows      []*wfv1.Workflow
		expectedStatus int
	}{
		{
			[]*wfv1.Workflow{veryOldWF},
			500,
		},
		{
			[]*wfv1.Workflow{newWF},
			200,
		},
		{
			[]*wfv1.Workflow{veryOldWF, newWF},
			500,
		},
		{
			[]*wfv1.Workflow{veryOldButReconciledWF},
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
