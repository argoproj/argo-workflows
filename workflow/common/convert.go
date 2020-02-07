package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func ConvertCronWorkflowToWorkflow(cronWf *wfv1.CronWorkflow) (*wfv1.Workflow, error) {
	newTypeMeta := metav1.TypeMeta{
		Kind:       workflow.WorkflowKind,
		APIVersion: cronWf.TypeMeta.APIVersion,
	}

	newObjectMeta := metav1.ObjectMeta{}
	newObjectMeta.GenerateName = cronWf.Name + "-"

	newObjectMeta.Labels = make(map[string]string)
	newObjectMeta.Labels[LabelKeyCronWorkflow] = cronWf.Name
	if instanceId, ok := cronWf.GetLabels()[LabelKeyControllerInstanceID]; ok {
		newObjectMeta.Labels[LabelKeyControllerInstanceID] = instanceId
	}

	wf := &wfv1.Workflow{
		TypeMeta:   newTypeMeta,
		ObjectMeta: newObjectMeta,
		Spec:       cronWf.Spec.WorkflowSpec,
	}
	wf.SetOwnerReferences(append(wf.GetOwnerReferences(), *metav1.NewControllerRef(cronWf, wfv1.SchemeGroupVersion.WithKind(workflow.CronWorkflowKind))))
	return wf, nil
}
