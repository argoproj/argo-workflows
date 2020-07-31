package common

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func ConvertCronWorkflowToWorkflow(cronWf *wfv1.CronWorkflow, scheduleTime time.Time) *wfv1.Workflow {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%v", cronWf.GetName(), scheduleTime.Unix()),
			Labels: map[string]string{
				LabelKeyCronWorkflow: cronWf.Name,
			},
			Annotations: make(map[string]string),
		},
		Spec: cronWf.Spec.WorkflowSpec,
	}
	if instanceId, ok := cronWf.GetLabels()[LabelKeyControllerInstanceID]; ok {
		wf.ObjectMeta.GetLabels()[LabelKeyControllerInstanceID] = instanceId
	}
	if cronWf.Spec.WorkflowMetadata != nil {
		for key, label := range cronWf.Spec.WorkflowMetadata.Labels {
			wf.Labels[key] = label
		}
		for key, annotation := range cronWf.Spec.WorkflowMetadata.Annotations {
			wf.Annotations[key] = annotation
		}
	}
	wf.SetOwnerReferences(append(wf.GetOwnerReferences(), *metav1.NewControllerRef(cronWf, wfv1.SchemeGroupVersion.WithKind(workflow.CronWorkflowKind))))
	intOrString := intstr.Parse(scheduleTime.Format(time.RFC3339))
	wf.Spec.Arguments.Parameters = append(wf.Spec.Arguments.Parameters, wfv1.Parameter{Name: "cronScheduleTime", Value: &intOrString})
	return wf
}

func NewWorkflowFromWorkflowTemplate(templateName string, workflowMetadata *metav1.ObjectMeta, clusterScope bool) *wfv1.Workflow {

	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: templateName + "-",
			Labels:       make(map[string]string),
			Annotations:  make(map[string]string),
		},
		Spec: wfv1.WorkflowSpec{
			WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{
				Name:         templateName,
				ClusterScope: clusterScope,
			},
		},
	}

	if workflowMetadata != nil {
		for key, label := range workflowMetadata.Labels {
			wf.Labels[key] = label
		}
		for key, annotation := range workflowMetadata.Annotations {
			wf.Annotations[key] = annotation
		}
	}

	if clusterScope {
		wf.Labels[LabelKeyClusterWorkflowTemplate] = templateName
	} else {
		wf.Labels[LabelKeyWorkflowTemplate] = templateName
	}
	return wf
}
