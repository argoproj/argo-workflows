package controller

import (
	"fmt"
	"sync"
	"time"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	workflowclient "github.com/argoproj/argo/workflow/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (wfc *WorkflowController) simulateRun(wfcl *workflowclient.WorkflowClient, wf *wfv1.Workflow) {
	if wf.Completed() {
		return
	}
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	wfCopy := wf.DeepCopyObject().(*wfv1.Workflow)
	wfCopy.Status = wfv1.WorkflowStatusCreated
	wfCopy, err := wfcl.UpdateWorkflow(wfCopy)
	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED %s status: %#v\n", wfCopy.ObjectMeta.Name, wfCopy.Status)
	}

	time.Sleep(3 * time.Second)
	wfCopy.Status = wfv1.WorkflowStatusRunning
	wfCopy, err = wfcl.UpdateWorkflow(wfCopy)
	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED %s status: %#v\n", wfCopy.ObjectMeta.Name, wfCopy.Status)
	}

	targetTmpl := wfCopy.GetTemplate(wfCopy.Target)
	status, err := wfc.executeTemplate(wfCopy, targetTmpl, nil, nil)

	wfCopy.Status = status
	wfCopy, err = wfcl.UpdateWorkflow(wfCopy)
	if err != nil {
		fmt.Printf("ERROR updating status: %v\n", err)
	} else {
		fmt.Printf("UPDATED %s status: %#v\n", wfCopy.ObjectMeta.Name, wfCopy.Status)
	}

}

func (wfc *WorkflowController) executeTemplate(wf *wfv1.Workflow, tmpl *wfv1.Template, args *wfv1.Arguments, wg *sync.WaitGroup) (string, error) {
	fmt.Printf("Executing %v, args: %#v\n", tmpl, args)
	if wg != nil {
		defer func() { wg.Done() }()
	}

	switch tmpl.Type {
	case wfv1.TypeContainer:
		fmt.Printf("Creating container: %s\n", tmpl.Name)
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-", wf.ObjectMeta.Name),
				Labels: map[string]string{
					"workflow": wf.ObjectMeta.Name,
				},
			},
			Spec: corev1.PodSpec{
				RestartPolicy: corev1.RestartPolicyNever,
				Containers:    []corev1.Container{*tmpl.Container},
			},
		}
		created, err := wfc.podCl.Create(&pod)
		if err != nil {
			fmt.Printf("Failed to create pod %v: %v\n", pod, err)
			return wfv1.WorkflowStatusFailed, err
		}
		fmt.Printf("Created pod: %v\n", created)
	case wfv1.TypeWorkflow:
		for i, stepGroup := range tmpl.Steps {
			var wg sync.WaitGroup
			for stepName, step := range stepGroup {
				wg.Add(1)
				targetTmpl := wf.GetTemplate(step.Template)
				fmt.Printf("Executing step[%d] %s\n", i, stepName)
				go wfc.executeTemplate(wf, targetTmpl, &step.Arguments, &wg)
			}
			wg.Wait()
			fmt.Printf("Completed %s step group %d\n", wf.Name, i)
		}
	default:
		return wfv1.WorkflowStatusFailed, fmt.Errorf("Unknown type: %s", tmpl.Type)
	}

	return wfv1.WorkflowStatusSuccess, nil
}
