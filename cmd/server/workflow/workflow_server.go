package workflow

import (
	"bufio"
	"fmt"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/server/auth"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/packer"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
)

type workflowServer struct {
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	//wfKubeService         *kubeService
}

func NewWorkflowServer(offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo) WorkflowServiceServer {
	return &workflowServer{
		offloadNodeStatusRepo: offloadNodeStatusRepo,
	}
}

func (s *workflowServer) CreateWorkflow(ctx context.Context, req *WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
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

func (s *workflowServer) GetWorkflow(ctx context.Context, req *WorkflowGetRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wfGetOption := metav1.GetOptions{}
	if req.GetOptions != nil {
		wfGetOption = *req.GetOptions
	}
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(req.Name, wfGetOption)
	if err != nil {
		return nil, err
	}

	if wf.Status.OffloadNodeStatus {
		offloaded, err := s.offloadNodeStatusRepo.Get(req.Name, req.Namespace)
		if err != nil {
			return nil, err
		}
		wf.Status.Nodes = offloaded.Status.Nodes
		wf.Status.CompressedNodes = offloaded.Status.CompressedNodes
	}
	err = packer.DecompressWorkflow(wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) ListWorkflows(ctx context.Context, req *WorkflowListRequest) (*v1alpha1.WorkflowList, error) {
	wfClient := auth.GetWfClient(ctx)

	var listOption = metav1.ListOptions{}
	if req.ListOptions != nil {
		listOption = *req.ListOptions
	}

	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).List(listOption)
	if err != nil {
		return nil, err
	}

	return wfList, nil
}

func (s *workflowServer) WatchWorkflows(req *WatchWorkflowsRequest, ws WorkflowService_WatchWorkflowsServer) error {
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

	for next := range watch.ResultChan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		wf := next.Object.(*v1alpha1.Workflow)
		err := packer.DecompressWorkflow(wf)
		logCtx := log.WithFields(log.Fields{"type": next.Type, "namespace": wf.Namespace, "workflowName": wf.Name})
		if err != nil {
			return err
		}
		if wf.Status.OffloadNodeStatus {
			offloaded, err := s.offloadNodeStatusRepo.Get(wf.Name, wf.Namespace)
			if err != nil {
				return err
			}
			wf.Status.Nodes = offloaded.Status.Nodes
		}
		logCtx.Debug("Sending event")
		err = ws.Send(&WorkflowWatchEvent{Type: string(next.Type), Object: wf})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *workflowServer) DeleteWorkflow(ctx context.Context, req *WorkflowDeleteRequest) (*WorkflowDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if wf.Status.OffloadNodeStatus {
		err = s.offloadNodeStatusRepo.Delete(req.Name, req.Namespace)
		if err != nil {
			return nil, err
		}
	}
	err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Delete(req.Name, &metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return &WorkflowDeleteResponse{}, nil
}

func (s *workflowServer) RetryWorkflow(ctx context.Context, req *WorkflowRetryRequest) (*v1alpha1.Workflow, error) {
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

func (s *workflowServer) ResubmitWorkflow(ctx context.Context, req *WorkflowResubmitRequest) (*v1alpha1.Workflow, error) {
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

func (s *workflowServer) ResumeWorkflow(ctx context.Context, req *WorkflowResumeRequest) (*v1alpha1.Workflow, error) {
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

func (s *workflowServer) SuspendWorkflow(ctx context.Context, req *WorkflowSuspendRequest) (*v1alpha1.Workflow, error) {
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

func (s *workflowServer) TerminateWorkflow(ctx context.Context, req *WorkflowTerminateRequest) (*v1alpha1.Workflow, error) {
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

func (s *workflowServer) LintWorkflow(ctx context.Context, req *WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))

	err := validate.ValidateWorkflow(wftmplGetter, req.Workflow, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}

	return req.Workflow, nil
}

func (s *workflowServer) PodLogs(req *WorkflowLogRequest, ws WorkflowService_PodLogsServer) error {
	kubeClient := auth.GetKubeClient(ws.Context())
	stream, err := kubeClient.CoreV1().Pods(req.Namespace).GetLogs(req.PodName, req.LogOptions).Stream()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		err = ws.Send(&LogEntry{Content: scanner.Text()})
		if err != nil {
			return err
		}
	}
	return nil
}
