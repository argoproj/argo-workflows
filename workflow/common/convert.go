package common

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// If `scheduleTime == 0`, then we assume that you are manually (i.e. `--from`). The name will be generated and
// `cronScheduleTime` parameter will not be set.
// If `scheduleTime > 0` then we assume this is being created on a cron schedule and the name will be deterministic AND
// the `cronScheduleTime` parameter will be set to its value (RFC3339).
func ConvertCronWorkflowToWorkflow(cronWf *wfv1.CronWorkflow, scheduleTime time.Time) *wfv1.Workflow {
	generateName := cronWf.Name + "-"
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: generateName,
			Labels: map[string]string{
				LabelKeyCronWorkflow: cronWf.Name,
			},
			Annotations: make(map[string]string),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cronWf, wfv1.SchemeGroupVersion.WithKind(workflow.CronWorkflowKind)),
			},
		},
		Spec: cronWf.Spec.WorkflowSpec,
	}
	if !scheduleTime.IsZero() {
		// truncate the time to 1m because we know that no more than one cron job can run per minute
		cronWf.SetName(fmt.Sprintf("%s%v", generateName, scheduleTime.Truncate(time.Minute).Unix()))
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
	if !scheduleTime.IsZero() {
		intOrString := intstr.Parse(scheduleTime.Format(time.RFC3339))
		wf.Spec.Arguments.Parameters = append(wf.Spec.Arguments.Parameters, wfv1.Parameter{Name: "cronScheduleTime", Value: &intOrString})
	}
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
