package casbin

import (
	"context"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type WorkflowTaskSetsEnforced struct {
	delegate v1alpha1.WorkflowTaskSetInterface
}

func (c WorkflowEnforcedInterface) WorkflowTaskSets(namespace string) v1alpha1.WorkflowTaskSetInterface {
	return &WorkflowTaskSetsEnforced{c.delegate.ArgoprojV1alpha1().WorkflowTaskSets(namespace)}
}

func (w WorkflowTaskSetsEnforced) Create(ctx context.Context, workflowTaskSet *wfv1.WorkflowTaskSet, opts metav1.CreateOptions) (*wfv1.WorkflowTaskSet, error) {
	panic("implement me")
}

func (w WorkflowTaskSetsEnforced) Update(ctx context.Context, workflowTaskSet *wfv1.WorkflowTaskSet, opts metav1.UpdateOptions) (*wfv1.WorkflowTaskSet, error) {
	panic("implement me")
}

func (w WorkflowTaskSetsEnforced) UpdateStatus(ctx context.Context, workflowTaskSet *wfv1.WorkflowTaskSet, opts metav1.UpdateOptions) (*wfv1.WorkflowTaskSet, error) {
	panic("implement me")
}

func (w WorkflowTaskSetsEnforced) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	panic("implement me")
}

func (w WorkflowTaskSetsEnforced) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("implement me")
}

func (w WorkflowTaskSetsEnforced) Get(ctx context.Context, name string, opts metav1.GetOptions) (*wfv1.WorkflowTaskSet, error) {
	panic("implement me")
}

func (w WorkflowTaskSetsEnforced) List(ctx context.Context, opts metav1.ListOptions) (*wfv1.WorkflowTaskSetList, error) {
	panic("implement me")
}

func (w WorkflowTaskSetsEnforced) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (w WorkflowTaskSetsEnforced) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *wfv1.WorkflowTaskSet, err error) {
	panic("implement me")
}