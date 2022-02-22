package cronworkflow

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
)

type cronWorkflowServiceServer struct {
	instanceIDService instanceid.Service
}

// NewCronWorkflowServer returns a new cronWorkflowServiceServer
func NewCronWorkflowServer(instanceIDService instanceid.Service) cronworkflowpkg.CronWorkflowServiceServer {
	return &cronWorkflowServiceServer{instanceIDService}
}

func (c *cronWorkflowServiceServer) LintCronWorkflow(ctx context.Context, req *cronworkflowpkg.LintCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	c.instanceIDService.Label(req.CronWorkflow)
	creator.Label(ctx, req.CronWorkflow)
	err := validate.ValidateCronWorkflow(wftmplGetter, cwftmplGetter, req.CronWorkflow)
	if err != nil {
		return nil, err
	}
	return req.CronWorkflow, nil
}

func (c *cronWorkflowServiceServer) ListCronWorkflows(ctx context.Context, req *cronworkflowpkg.ListCronWorkflowsRequest) (*v1alpha1.CronWorkflowList, error) {
	options := &metav1.ListOptions{}
	if req.ListOptions != nil {
		options = req.ListOptions
	}
	c.instanceIDService.With(options)
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).List(ctx, *options)
}

func (c *cronWorkflowServiceServer) CreateCronWorkflow(ctx context.Context, req *cronworkflowpkg.CreateCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	wfClient := auth.GetWfClient(ctx)
	if req.CronWorkflow == nil {
		return nil, fmt.Errorf("cron workflow was not found in the request body")
	}
	c.instanceIDService.Label(req.CronWorkflow)
	creator.Label(ctx, req.CronWorkflow)
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	err := validate.ValidateCronWorkflow(wftmplGetter, cwftmplGetter, req.CronWorkflow)
	if err != nil {
		return nil, err
	}
	return wfClient.ArgoprojV1alpha1().CronWorkflows(req.Namespace).Create(ctx, req.CronWorkflow, metav1.CreateOptions{})
}

func (c *cronWorkflowServiceServer) GetCronWorkflow(ctx context.Context, req *cronworkflowpkg.GetCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	options := metav1.GetOptions{}
	if req.GetOptions != nil {
		options = *req.GetOptions
	}
	return c.getCronWorkflowAndValidate(ctx, req.Namespace, req.Name, options)
}

func (c *cronWorkflowServiceServer) UpdateCronWorkflow(ctx context.Context, req *cronworkflowpkg.UpdateCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	wfClient := auth.GetWfClient(ctx)
	_, err := c.getCronWorkflowAndValidate(ctx, req.Namespace, req.CronWorkflow.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	if err := validate.ValidateCronWorkflow(wftmplGetter, cwftmplGetter, req.CronWorkflow); err != nil {
		return nil, err
	}
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Update(ctx, req.CronWorkflow, metav1.UpdateOptions{})
}

func (c *cronWorkflowServiceServer) DeleteCronWorkflow(ctx context.Context, req *cronworkflowpkg.DeleteCronWorkflowRequest) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	_, err := c.getCronWorkflowAndValidate(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	opts := metav1.DeleteOptions{}
	if req.DeleteOptions != nil {
		opts = *req.DeleteOptions
	}

	err = auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Delete(ctx, req.Name, opts)
	if err != nil {
		return nil, err
	}
	return &cronworkflowpkg.CronWorkflowDeletedResponse{}, nil
}

func (c *cronWorkflowServiceServer) ResumeCronWorkflow(ctx context.Context, req *cronworkflowpkg.CronWorkflowResumeRequest) (*v1alpha1.CronWorkflow, error) {
	return setCronWorkflowSuspend(ctx, false, req.Namespace, req.Name)
}

func (c *cronWorkflowServiceServer) SuspendCronWorkflow(ctx context.Context, req *cronworkflowpkg.CronWorkflowSuspendRequest) (*v1alpha1.CronWorkflow, error) {
	return setCronWorkflowSuspend(ctx, true, req.Namespace, req.Name)
}

func setCronWorkflowSuspend(ctx context.Context, setTo bool, namespace, name string) (*v1alpha1.CronWorkflow, error) {
	data, err := json.Marshal(map[string]interface{}{"spec": map[string]interface{}{"suspend": setTo}})
	if err != nil {
		return nil, fmt.Errorf("failed to marshall cron workflow patch data: %w", err)
	}
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(namespace).Patch(ctx, name, types.MergePatchType, data, metav1.PatchOptions{})
}

func (c *cronWorkflowServiceServer) getCronWorkflowAndValidate(ctx context.Context, namespace string, name string, options metav1.GetOptions) (*v1alpha1.CronWorkflow, error) {
	wfClient := auth.GetWfClient(ctx)
	cronWf, err := wfClient.ArgoprojV1alpha1().CronWorkflows(namespace).Get(ctx, name, options)
	if err != nil {
		return nil, err
	}
	err = c.instanceIDService.Validate(cronWf)
	if err != nil {
		return nil, err
	}
	return cronWf, nil
}
