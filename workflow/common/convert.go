package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func ConvertCronWorkflowToWorkflow(cronWf *wfv1.CronWorkflow) *wfv1.Workflow {
	wf := toWorkflow(cronWf.TypeMeta, cronWf.ObjectMeta, cronWf.Spec.WorkflowSpec)
	wf.Labels[LabelKeyCronWorkflow] = cronWf.Name
	if cronWf.Spec.WorkflowMetadata != nil {
		for key, label := range cronWf.Spec.WorkflowMetadata.Labels {
			wf.Labels[key] = label
		}

		if len(cronWf.Spec.WorkflowMetadata.Annotations) > 0 {
			wf.Annotations = make(map[string]string)
			for key, annotation := range cronWf.Spec.WorkflowMetadata.Annotations {
				wf.Annotations[key] = annotation
			}
		}
	}
	wf.SetOwnerReferences(append(wf.GetOwnerReferences(), *metav1.NewControllerRef(cronWf, wfv1.SchemeGroupVersion.WithKind(workflow.CronWorkflowKind))))
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

func toWorkflow(typeMeta metav1.TypeMeta, objectMeta metav1.ObjectMeta, spec wfv1.WorkflowSpec) *wfv1.Workflow {
	wf := &wfv1.Workflow{
		TypeMeta: metav1.TypeMeta{
			Kind:       workflow.WorkflowKind,
			APIVersion: typeMeta.APIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: objectMeta.GetName() + "-",
			Labels:       make(map[string]string),
			Annotations:  make(map[string]string),
		},
		Spec: spec,
	}

	if instanceId, ok := objectMeta.GetLabels()[LabelKeyControllerInstanceID]; ok {
		wf.ObjectMeta.GetLabels()[LabelKeyControllerInstanceID] = instanceId
	}

	return wf
}
