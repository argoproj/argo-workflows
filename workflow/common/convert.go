package common

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func ConvertCronWorkflowToWorkflow(cronWf *wfv1.CronWorkflow) (*wfv1.Workflow, error) {
	err := ConvertToTemplatedWorkflow(cronWf)
	if err != nil {
		return nil, err
	}
	wf := toWorkflow(cronWf.TypeMeta, cronWf.ObjectMeta, cronWf.Spec.Template)
	wf.Labels[LabelKeyCronWorkflow] = cronWf.Name
	wf.SetOwnerReferences(append(wf.GetOwnerReferences(), *metav1.NewControllerRef(cronWf, wfv1.SchemeGroupVersion.WithKind(workflow.CronWorkflowKind))))
	return wf, nil
}

func ConvertToTemplatedWorkflow(cronWf *wfv1.CronWorkflow) error {
	if cronWf.Spec.WorkflowSpec != nil && cronWf.Spec.Template != nil {
		return fmt.Errorf("cannot use both CronWorkflow.spec.workflowSpec and CronWorkflow.spec.template to specify a Workflow to run. Please use only CronWorkflow.spec.template instead")
	}
	if cronWf.Spec.WorkflowMetadata != nil && cronWf.Spec.Template != nil {
		return fmt.Errorf("cannot use both CronWorkflow.spec.workflowMetadata and CronWorkflow.spec.template to specify a Workflow to run. Please use only CronWorkflow.spec.template instead")
	}
	if cronWf.Spec.Template != nil {
		return nil
	}
	cronWf.Spec.Template = &wfv1.Workflow{
		ObjectMeta: *cronWf.Spec.WorkflowMetadata,
		Spec:       *cronWf.Spec.WorkflowSpec,
	}
	cronWf.Spec.WorkflowMetadata = nil
	cronWf.Spec.WorkflowSpec = nil
	return nil
}

func NewWorkflowFromWorkflowTemplate(templateName string, clusterScope bool) *wfv1.Workflow {
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
	return wf
}

func toWorkflow(typeMeta metav1.TypeMeta, objectMeta metav1.ObjectMeta, template *wfv1.Workflow) *wfv1.Workflow {
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
		Spec: template.Spec,
	}

	if instanceId, ok := objectMeta.GetLabels()[LabelKeyControllerInstanceID]; ok {
		wf.ObjectMeta.GetLabels()[LabelKeyControllerInstanceID] = instanceId
	}

	for key, label := range template.ObjectMeta.Labels {
		wf.Labels[key] = label
	}

	if len(template.ObjectMeta.Annotations) > 0 {
		wf.Annotations = make(map[string]string)
		for key, label := range template.ObjectMeta.Annotations {
			wf.Annotations[key] = label
		}
	}

	return wf
}
