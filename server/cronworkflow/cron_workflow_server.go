package cronworkflow

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cronworkflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	servertypes "github.com/argoproj/argo-workflows/v4/server/types"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	"github.com/argoproj/argo-workflows/v4/workflow/creator"
	"github.com/argoproj/argo-workflows/v4/workflow/validate"

	sutils "github.com/argoproj/argo-workflows/v4/server/utils"
)

type cronWorkflowServiceServer struct {
	instanceIDService instanceid.Service
	wftmplStore       servertypes.WorkflowTemplateStore
	cwftmplStore      servertypes.ClusterWorkflowTemplateStore
	wfDefaults        *v1alpha1.Workflow
}

// NewCronWorkflowServer returns a new cronWorkflowServiceServer
func NewCronWorkflowServer(instanceIDService instanceid.Service, wftmplStore servertypes.WorkflowTemplateStore, cwftmplStore servertypes.ClusterWorkflowTemplateStore, wfDefaults *v1alpha1.Workflow) cronworkflowpkg.CronWorkflowServiceServer {
	return &cronWorkflowServiceServer{instanceIDService, wftmplStore, cwftmplStore, wfDefaults}
}

func (c *cronWorkflowServiceServer) LintCronWorkflow(ctx context.Context, req *cronworkflowpkg.LintCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	wftmplGetter := c.wftmplStore.Getter(ctx, req.Namespace)
	cwftmplGetter := c.cwftmplStore.Getter(ctx)
	c.instanceIDService.Label(req.CronWorkflow)
	creator.LabelCreator(ctx, req.CronWorkflow)
	err := validate.ValidateCronWorkflow(ctx, wftmplGetter, cwftmplGetter, req.CronWorkflow, c.wfDefaults)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	return req.CronWorkflow, nil
}

func (c *cronWorkflowServiceServer) ListCronWorkflows(ctx context.Context, req *cronworkflowpkg.ListCronWorkflowsRequest) (*v1alpha1.CronWorkflowList, error) {
	options := &metav1.ListOptions{}
	if req.ListOptions != nil {
		options = req.ListOptions
	}
	c.instanceIDService.With(options)
	cronWfList, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).List(ctx, *options)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return cronWfList, nil
}

func (c *cronWorkflowServiceServer) CreateCronWorkflow(ctx context.Context, req *cronworkflowpkg.CreateCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	wfClient := auth.GetWfClient(ctx)
	if req.CronWorkflow == nil {
		return nil, sutils.ToStatusError(fmt.Errorf("cron workflow was not found in the request body"), codes.NotFound)
	}
	c.instanceIDService.Label(req.CronWorkflow)
	creator.LabelCreator(ctx, req.CronWorkflow)
	wftmplGetter := c.wftmplStore.Getter(ctx, req.Namespace)
	cwftmplGetter := c.cwftmplStore.Getter(ctx)
	err := validate.ValidateCronWorkflow(ctx, wftmplGetter, cwftmplGetter, req.CronWorkflow, c.wfDefaults)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	crWf, err := wfClient.ArgoprojV1alpha1().CronWorkflows(req.Namespace).Create(ctx, req.CronWorkflow, metav1.CreateOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return crWf, nil
}

func (c *cronWorkflowServiceServer) GetCronWorkflow(ctx context.Context, req *cronworkflowpkg.GetCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	options := metav1.GetOptions{}
	if req.GetOptions != nil {
		options = *req.GetOptions
	}
	return c.getCronWorkflowAndValidate(ctx, req.Namespace, req.Name, options)
}

func (c *cronWorkflowServiceServer) UpdateCronWorkflow(ctx context.Context, req *cronworkflowpkg.UpdateCronWorkflowRequest) (*v1alpha1.CronWorkflow, error) {
	_, err := c.getCronWorkflowAndValidate(ctx, req.Namespace, req.CronWorkflow.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	creator.LabelActor(ctx, req.CronWorkflow, creator.ActionUpdate)
	wftmplGetter := c.wftmplStore.Getter(ctx, req.Namespace)
	cwftmplGetter := c.cwftmplStore.Getter(ctx)
	if err := validate.ValidateCronWorkflow(ctx, wftmplGetter, cwftmplGetter, req.CronWorkflow, c.wfDefaults); err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	crWf, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Update(ctx, req.CronWorkflow, metav1.UpdateOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return crWf, nil
}

func (c *cronWorkflowServiceServer) DeleteCronWorkflow(ctx context.Context, req *cronworkflowpkg.DeleteCronWorkflowRequest) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	_, err := c.getCronWorkflowAndValidate(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	opts := metav1.DeleteOptions{}
	if req.DeleteOptions != nil {
		opts = *req.DeleteOptions
	}

	err = auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(req.Namespace).Delete(ctx, req.Name, opts)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return &cronworkflowpkg.CronWorkflowDeletedResponse{}, nil
}

func (c *cronWorkflowServiceServer) ResumeCronWorkflow(ctx context.Context, req *cronworkflowpkg.CronWorkflowResumeRequest) (*v1alpha1.CronWorkflow, error) {
	crWf, err := setCronWorkflowSuspend(ctx, false, req.Namespace, req.Name)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return crWf, nil
}

func (c *cronWorkflowServiceServer) SuspendCronWorkflow(ctx context.Context, req *cronworkflowpkg.CronWorkflowSuspendRequest) (*v1alpha1.CronWorkflow, error) {
	crWf, err := setCronWorkflowSuspend(ctx, true, req.Namespace, req.Name)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return crWf, nil
}

func setCronWorkflowSuspend(ctx context.Context, setTo bool, namespace, name string) (*v1alpha1.CronWorkflow, error) {
	data, err := json.Marshal(map[string]interface{}{"spec": map[string]interface{}{"suspend": setTo}})
	if err != nil {
		return nil, sutils.ToStatusError(fmt.Errorf("failed to marshall cron workflow patch data: %w", err), codes.Internal)
	}
	cronWfs, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().CronWorkflows(namespace).Patch(ctx, name, types.MergePatchType, data, metav1.PatchOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return cronWfs, nil
}

func (c *cronWorkflowServiceServer) getCronWorkflowAndValidate(ctx context.Context, namespace string, name string, options metav1.GetOptions) (*v1alpha1.CronWorkflow, error) {
	wfClient := auth.GetWfClient(ctx)
	cronWf, err := wfClient.ArgoprojV1alpha1().CronWorkflows(namespace).Get(ctx, name, options)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	err = c.instanceIDService.Validate(cronWf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	return cronWf, nil
}
