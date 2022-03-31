package executor

import (
	"context"
	"encoding/json"
	"os"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (we *WorkflowExecutor) upsertTaskResult(ctx context.Context, result wfv1.NodeResult) error {
	err := we.createTaskResult(ctx, result)
	if apierr.IsAlreadyExists(err) {
		return we.patchTaskResult(ctx, result)
	}
	return err
}

func (we *WorkflowExecutor) patchTaskResult(ctx context.Context, result wfv1.NodeResult) error {
	data, err := json.Marshal(&wfv1.WorkflowTaskResult{NodeResult: result})
	if err != nil {
		return err
	}
	_, err = we.taskResultClient.Patch(ctx,
		we.nodeId,
		types.MergePatchType,
		data,
		metav1.PatchOptions{},
	)
	return err
}

func (we *WorkflowExecutor) createTaskResult(ctx context.Context, result wfv1.NodeResult) error {
	taskResult := &wfv1.WorkflowTaskResult{
		TypeMeta: metav1.TypeMeta{
			APIVersion: workflow.APIVersion,
			Kind:       workflow.WorkflowTaskResultKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   we.nodeId,
			Labels: map[string]string{common.LabelKeyWorkflow: we.workflow},
		},
		NodeResult: result,
	}
	taskResult.SetOwnerReferences(
		[]metav1.OwnerReference{
			{
				APIVersion: "v1",
				Kind:       "pods",
				Name:       we.PodName,
				UID:        we.podUID,
			},
		})

	if v := os.Getenv(common.EnvVarInstanceID); v != "" {
		taskResult.Labels[common.LabelKeyControllerInstanceID] = v
	}
	_, err := we.taskResultClient.Create(ctx,
		taskResult,
		metav1.CreateOptions{},
	)
	return err
}
