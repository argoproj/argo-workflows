package workflow

import (
	"fmt"
	"reflect"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/persist/sqldb"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/logs"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/packer"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
)

type workflowServer struct {
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
}

func NewWorkflowServer(offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo) workflowpkg.WorkflowServiceServer {
	return &workflowServer{
		offloadNodeStatusRepo: offloadNodeStatusRepo,
	}
}

func (s *workflowServer) CreateWorkflow(ctx context.Context, req *workflowpkg.WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	if req.Workflow == nil {
		return nil, fmt.Errorf("workflow body not specified")
	}

	if req.Workflow.Namespace == "" {
		req.Workflow.Namespace = req.Namespace
	}

	if req.InstanceID != "" {
		labels := req.Workflow.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[common.LabelKeyControllerInstanceID] = req.InstanceID
		req.Workflow.SetLabels(labels)
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))

	err := validate.ValidateWorkflow(wftmplGetter, req.Workflow, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}

	// if we are doing a normal dryRun, just return the workflow un-altered
	if req.CreateOptions != nil && len(req.CreateOptions.DryRun) > 0 {
		return req.Workflow, nil
	}
	if req.ServerDryRun {
		return util.CreateServerDryRun(req.Workflow, wfClient)
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Create(req.Workflow)

	if err != nil {
		log.Errorf("Create request is failed. Error: %s", err)
		return nil, err

	}
	return wf, nil
}

func (s *workflowServer) GetWorkflow(ctx context.Context, req *workflowpkg.WorkflowGetRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wfGetOption := metav1.GetOptions{}
	if req.GetOptions != nil {
		wfGetOption = *req.GetOptions
	}
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(req.Name, wfGetOption)
	if err != nil {
		return nil, err
	}

	if wf.Status.IsOffloadNodeStatus() && s.offloadNodeStatusRepo.IsEnabled() {
		offloadedNodes, err := s.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
		if err != nil {
			return nil, err
		}
		wf.Status.Nodes = offloadedNodes
	}
	err = packer.DecompressWorkflow(wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) ListWorkflows(ctx context.Context, req *workflowpkg.WorkflowListRequest) (*v1alpha1.WorkflowList, error) {
	wfClient := auth.GetWfClient(ctx)

	var listOption = metav1.ListOptions{}
	if req.ListOptions != nil {
		listOption = *req.ListOptions
	}

	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).List(listOption)
	if err != nil {
		return nil, err
	}
	if s.offloadNodeStatusRepo.IsEnabled() {
		offloadedNodes, err := s.offloadNodeStatusRepo.List(req.Namespace)
		if err != nil {
			return nil, err
		}
		for i, wf := range wfList.Items {
			if wf.Status.IsOffloadNodeStatus() {
				wfList.Items[i].Status.Nodes = offloadedNodes[sqldb.UUIDVersion{UID: string(wf.UID), Version: wf.GetOffloadNodeStatusVersion()}]
			}
		}
	}

	return &v1alpha1.WorkflowList{Items: wfList.Items}, nil
}

func (s *workflowServer) WatchWorkflows(req *workflowpkg.WatchWorkflowsRequest, ws workflowpkg.WorkflowService_WatchWorkflowsServer) error {
	wfClient := auth.GetWfClient(ws.Context())
	opts := metav1.ListOptions{}
	if req.ListOptions != nil {
		opts = *req.ListOptions
	}
	watch, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Watch(opts)
	if err != nil {
		return err
	}
	defer watch.Stop()
	ctx := ws.Context()

	log.Debug("Piping events to channel")

	for next := range watch.ResultChan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		gvk := next.Object.GetObjectKind().GroupVersionKind().String()
		logCtx := log.WithFields(log.Fields{"type": next.Type, "objectKind": gvk})
		logCtx.Debug("Received event")
		wf, ok := next.Object.(*v1alpha1.Workflow)
		if !ok {
			return fmt.Errorf("watch object was not a workflow %v", reflect.TypeOf(next.Object))
		}
		err := packer.DecompressWorkflow(wf)
		if err != nil {
			return err
		}
		if wf.Status.IsOffloadNodeStatus() && s.offloadNodeStatusRepo.IsEnabled() {
			offloadedNodes, err := s.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
			if err != nil {
				return err
			}
			wf.Status.Nodes = offloadedNodes
		}
		logCtx.Debug("Sending event")
		err = ws.Send(&workflowpkg.WorkflowWatchEvent{Type: string(next.Type), Object: wf})
		if err != nil {
			return err
		}

	}

	return nil
}

func (s *workflowServer) DeleteWorkflow(ctx context.Context, req *workflowpkg.WorkflowDeleteRequest) (*workflowpkg.WorkflowDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)

	err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Delete(req.Name, &metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return &workflowpkg.WorkflowDeleteResponse{}, nil
}

func (s *workflowServer) RetryWorkflow(ctx context.Context, req *workflowpkg.WorkflowRetryRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	wf, err = util.RetryWorkflow(kubeClient, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) ResubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowResubmitRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	newWF, err := util.FormulateResubmitWorkflow(wf, req.Memoized)
	if err != nil {
		return nil, err
	}

	created, err := util.SubmitWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wfClient, req.Namespace, newWF, &util.SubmitOpts{})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *workflowServer) ResumeWorkflow(ctx context.Context, req *workflowpkg.WorkflowResumeRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	err := util.ResumeWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), req.Name)
	if err != nil {
		log.Warnf("Failed to resume %s: %+v", req.Name, err)
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *workflowServer) SuspendWorkflow(ctx context.Context, req *workflowpkg.WorkflowSuspendRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	err := util.SuspendWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), req.Name)
	if err != nil {
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *workflowServer) TerminateWorkflow(ctx context.Context, req *workflowpkg.WorkflowTerminateRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	err := util.TerminateWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), req.Name)
	if err != nil {
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) LintWorkflow(ctx context.Context, req *workflowpkg.WorkflowLintRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))

	err := validate.ValidateWorkflow(wftmplGetter, req.Workflow, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}

	return req.Workflow, nil
}

func (s *workflowServer) PodLogs(req *workflowpkg.WorkflowLogRequest, ws workflowpkg.WorkflowService_PodLogsServer) error {
	ctx := ws.Context()
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)
	logger, err := logs.NewWorkflowLogger(ctx, wfClient, kubeClient, req, ws)
	if err != nil {
		return err
	}
	logger.Run(ctx)
	return nil
}
