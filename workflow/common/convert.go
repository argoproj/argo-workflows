package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func ConvertCronWorkflowToWorkflow(cronWf *wfv1.CronWorkflow) *wfv1.Workflow {
	wf := toWorkflow(cronWf.TypeMeta, cronWf.ObjectMeta, cronWf.Spec.WorkflowSpec, cronWf.Spec.WorkflowMetadata)
	wf.Labels[LabelKeyCronWorkflow] = cronWf.Name
	wf.SetOwnerReferences(append(wf.GetOwnerReferences(), *metav1.NewControllerRef(cronWf, wfv1.SchemeGroupVersion.WithKind(workflow.CronWorkflowKind))))
	return wf
}

func ConvertWorkflowTemplateToWorkflow(template *wfv1.WorkflowTemplate) *wfv1.Workflow {
	wf := toWorkflow(template.TypeMeta, template.ObjectMeta, template.Spec.WorkflowSpec, template.Spec.WorkflowMetadata)
	wf.Labels[LabelKeyWorkflowTemplate] = template.ObjectMeta.Name
	return wf
}

func ConvertClusterWorkflowTemplateToWorkflow(template *wfv1.ClusterWorkflowTemplate) *wfv1.Workflow {
	wf := toWorkflow(template.TypeMeta, template.ObjectMeta, template.Spec.WorkflowSpec, template.Spec.WorkflowMetadata)
	wf.Labels[LabelKeyClusterWorkflowTemplate] = template.ObjectMeta.Name
	return wf
}

func toWorkflow(typeMeta metav1.TypeMeta, objectMeta metav1.ObjectMeta, spec wfv1.WorkflowSpec, workflowMetadata *metav1.ObjectMeta) *wfv1.Workflow {
	if workflowMetadata == nil {
		workflowMetadata = &metav1.ObjectMeta{}
	}
	if workflowMetadata.Annotations == nil {
		workflowMetadata.Annotations = make(map[string]string)
	}
	if workflowMetadata.Labels == nil {
		workflowMetadata.Labels = make(map[string]string)
	}
	wf := &wfv1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       workflow.WorkflowKind,
			APIVersion: typeMeta.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: objectMeta.GetName() + "-",
			Labels:       workflowMetadata.Labels,
			Annotations:  workflowMetadata.Annotations,
		},
		Spec: spec,
	}

	if instanceId, ok := objectMeta.GetLabels()[LabelKeyControllerInstanceID]; ok {
		wf.ObjectMeta.GetLabels()[LabelKeyControllerInstanceID] = instanceId
	}

	return wf
}
