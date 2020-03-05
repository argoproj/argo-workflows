package cronworkflow

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
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
	err := validate.ValidateCronWorkflow(wftmplGetter, req.CronWorkflow)
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
	options = c.withInstanceID(options)
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).List(options)
}

func (c *cronWorkflowServiceServer) CreateCronWorkflow(ctx context.Context, req *cronworkflowpkg.CreateCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	if len(c.instanceID) > 0 {
		labels := req.CronWorkflow.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[common.LabelKeyControllerInstanceID] = c.instanceID
		req.CronWorkflow.SetLabels(labels)
	}
	return auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Create(req.CronWorkflow)
}

func (c *cronWorkflowServiceServer) GetCronWorkflow(ctx context.Context, req *cronworkflowpkg.GetCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	options := metav1.GetOptions{}
	if req.GetOptions != nil {
		options = *req.GetOptions
	}
	cronWf, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Get(req.Name, options)
	if err != nil {
		return nil, err
	}
	err = c.validateInstanceID(cronWf)
	if err != nil {
		return nil, err
	}
	return cronWf, nil
}

func (c *cronWorkflowServiceServer) UpdateCronWorkflow(ctx context.Context, req *cronworkflowpkg.UpdateCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	wfClient := auth.GetWfClient(ctx)
	cronWf, err := wfClient.ArgoprojV1alpha1().CronWorkflows(req.Namespace).Get(req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = c.validateInstanceID(cronWf)
	if err != nil {
		return nil, err
	}
	cronWf, err = wfClient.ArgoprojV1alpha1().CronWorkflows(req.Namespace).Update(req.CronWorkflow)
	if err != nil {
		return nil, err
	}
	return cronWf, nil
}

func (c *cronWorkflowServiceServer) DeleteCronWorkflow(ctx context.Context, req *cronworkflowpkg.DeleteCronWorkflowRequest) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	wfClient := auth.GetWfClient(ctx)
	cronWf, err := wfClient.ArgoprojV1alpha1().CronWorkflows(req.Namespace).Get(req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = c.validateInstanceID(cronWf)
	if err != nil {
		return nil, err
	}
	err = wfClient.ArgoprojV1alpha1().CronWorkflows(req.Namespace).Delete(req.Name, req.DeleteOptions)
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

func (c *cronWorkflowServiceServer) validateInstanceID(cronWf *v1alpha1.CronWorkflow) error {
	if len(c.instanceID) == 0 {
		if len(cronWf.Labels) == 0 {
			return nil
		}
		if _, ok := cronWf.Labels[common.LabelKeyControllerInstanceID]; !ok {
			return nil
		}
	} else if len(cronWf.Labels) > 0 {
		if val, ok := cronWf.Labels[common.LabelKeyControllerInstanceID]; ok {
			if val == c.instanceID {
				return nil
			}
		}
	}
	return fmt.Errorf("the CronWorkflow is not managed by current Argo server")
}
