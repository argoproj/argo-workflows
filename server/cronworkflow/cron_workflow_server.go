package cronworkflow

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/resource"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/validate"
)

type cronWorkflowServiceServer struct {
	instanceID string
}

// NewCronWorkflowServer returns a new cronWorkflowServiceServer
func NewCronWorkflowServer(instanceID string) cronworkflowpkg.CronWorkflowServiceServer {
	return &cronWorkflowServiceServer{instanceID: instanceID}
}

func (c *cronWorkflowServiceServer) LintCronWorkflow(ctx context.Context, req *cronworkflowpkg.LintCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	err := validate.ValidateCronWorkflow(wftmplGetter, cwftmplGetter, req.CronWorkflow)
	if err != nil {
		return nil, err
	}
	return req.CronWorkflow, nil
}

func (c *cronWorkflowServiceServer) ListCronWorkflows(ctx context.Context, req *cronworkflowpkg.ListCronWorkflowsRequest) (*v1alpha1.CronWorkflowList, error) {
	options := metav1.ListOptions{}
	if req.ListOptions != nil {
		options = *req.ListOptions
	}
	optsWithInstanceId := c.withInstanceID(options)
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).List(optsWithInstanceId)
}

func (c *cronWorkflowServiceServer) CreateCronWorkflow(ctx context.Context, req *cronworkflowpkg.CreateCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	wfClient := auth.GetWfClient(ctx)
	if req.CronWorkflow == nil {
		return nil, fmt.Errorf("cron workflow was not found in the request body")
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	err := validate.ValidateCronWorkflow(wftmplGetter, cwftmplGetter, req.CronWorkflow)
	if err != nil {
		return nil, err
	}

	c.setInstanceID(req.CronWorkflow)
	c.setCreator(ctx, req.CronWorkflow)

	return wfClient.ArgoprojV1alpha1().CronWorkflows(req.Namespace).Create(req.CronWorkflow)
}

func (c *cronWorkflowServiceServer) setInstanceID(obj metav1.Object) {
	resource.Label(obj, common.LabelKeyControllerInstanceID, c.instanceID)
}

func (c *cronWorkflowServiceServer) setCreator(ctx context.Context, obj metav1.Object) {
	resource.Label(obj, common.LabelKeyControllerCreator, auth.GetUser(ctx).Name)
}
func (c *cronWorkflowServiceServer) GetCronWorkflow(ctx context.Context, req *cronworkflowpkg.GetCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	options := metav1.GetOptions{}
	if req.GetOptions != nil {
		options = *req.GetOptions
	}
	return c.getCronWorkflow(ctx, req.Namespace, req.Name, options)
}

func (c *cronWorkflowServiceServer) UpdateCronWorkflow(ctx context.Context, req *cronworkflowpkg.UpdateCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	_, err := c.getCronWorkflow(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Update(req.CronWorkflow)
}

func (c *cronWorkflowServiceServer) DeleteCronWorkflow(ctx context.Context, req *cronworkflowpkg.DeleteCronWorkflowRequest) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	_, err := c.getCronWorkflow(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Delete(req.Name, req.DeleteOptions)
	if err != nil {
		return nil, err
	}
	return &cronworkflowpkg.CronWorkflowDeletedResponse{}, nil
}

func (c *cronWorkflowServiceServer) withInstanceID(opt metav1.ListOptions) metav1.ListOptions {
	if len(opt.LabelSelector) > 0 {
		opt.LabelSelector += ","
	}
	if len(c.instanceID) == 0 {
		opt.LabelSelector += fmt.Sprintf("!%s", common.LabelKeyControllerInstanceID)
		return opt
	}
	opt.LabelSelector += fmt.Sprintf("%s=%s", common.LabelKeyControllerInstanceID, c.instanceID)
	return opt
}

func (c *cronWorkflowServiceServer) getCronWorkflow(ctx context.Context, namespace string, name string, options metav1.GetOptions) (*v1alpha1.CronWorkflow, error) {
	wfClient := auth.GetWfClient(ctx)
	cronWf, err := wfClient.ArgoprojV1alpha1().CronWorkflows(namespace).Get(name, options)
	if err != nil {
		return nil, err
	}
	ok := c.validateInstanceID(cronWf)
	if !ok {
		return nil, fmt.Errorf("CronWorkflow '%s' is not managed by the current Argo server", cronWf.Name)
	}
	return cronWf, nil
}

func (c *cronWorkflowServiceServer) validateInstanceID(cronWf *v1alpha1.CronWorkflow) bool {
	if len(c.instanceID) == 0 {
		if len(cronWf.Labels) == 0 {
			return true
		}
		if _, ok := cronWf.Labels[common.LabelKeyControllerInstanceID]; !ok {
			return true
		}
	} else if len(cronWf.Labels) > 0 {
		if val, ok := cronWf.Labels[common.LabelKeyControllerInstanceID]; ok {
			if val == c.instanceID {
				return true
			}
		}
	}
	return false
}
