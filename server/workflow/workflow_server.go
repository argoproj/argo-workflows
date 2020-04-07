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
	instanceID            string
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
}

// NewWorkflowServer returns a new workflowServer
func NewWorkflowServer(instanceID string, offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo) workflowpkg.WorkflowServiceServer {
	return &workflowServer{
		instanceID:            instanceID,
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

	if len(req.InstanceID) > 0 || len(s.instanceID) > 0 {
		labels := req.Workflow.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		if len(req.InstanceID) > 0 {
			labels[common.LabelKeyControllerInstanceID] = req.InstanceID
		} else {
			labels[common.LabelKeyControllerInstanceID] = s.instanceID
		}
		req.Workflow.SetLabels(labels)
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))

	_, err := validate.ValidateWorkflow(wftmplGetter, req.Workflow, validate.ValidateOpts{})
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
	wfGetOption := metav1.GetOptions{}
	if req.GetOptions != nil {
		wfGetOption = *req.GetOptions
	}
	wf, err := s.getWorkflow(ctx, req.Namespace, req.Name, wfGetOption)
	if err != nil {
		return nil, err
	}

	if wf.Status.IsOffloadNodeStatus() {
		if s.offloadNodeStatusRepo.IsEnabled() {
			offloadedNodes, err := s.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
			if err != nil {
				return nil, err
			}
			wf.Status.Nodes = offloadedNodes
		} else {
			log.WithFields(log.Fields{"namespace": wf.Namespace, "name": wf.Name}).Warn(sqldb.OffloadNodeStatusDisabledWarning)
		}
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

	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).List(s.withInstanceID(listOption))
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
				if s.offloadNodeStatusRepo.IsEnabled() {
					wfList.Items[i].Status.Nodes = offloadedNodes[sqldb.UUIDVersion{UID: string(wf.UID), Version: wf.GetOffloadNodeStatusVersion()}]
				} else {
					log.WithFields(log.Fields{"namespace": wf.Namespace, "name": wf.Name}).Warn("Workflow has offloaded nodes, but offloading has been disabled")
				}
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
	watch, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Watch(s.withInstanceID(opts))
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
		log.Debug("Received event")
		wf, ok := next.Object.(*v1alpha1.Workflow)
		if !ok {
			return fmt.Errorf("watch object was not a workflow %v", reflect.TypeOf(next.Object))
		}
		logCtx := log.WithFields(log.Fields{"workflow": wf.Name, "type": next.Type, "phase": wf.Status.Phase})
		err := packer.DecompressWorkflow(wf)
		if err != nil {
			return err
		}
		if wf.Status.IsOffloadNodeStatus() {
			if s.offloadNodeStatusRepo.IsEnabled() {
				offloadedNodes, err := s.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
				if err != nil {
					return err
				}
				wf.Status.Nodes = offloadedNodes
			} else {
				log.WithFields(log.Fields{"namespace": wf.Namespace, "name": wf.Name}).Warn(sqldb.OffloadNodeStatusDisabledWarning)
			}
		}
		logCtx.Debug("Sending event")
		err = ws.Send(&workflowpkg.WorkflowWatchEvent{Type: string(next.Type), Object: wf})
		if err != nil {
			return err
		}

	}

	log.Debug("Result channel done")

	return nil
}

func (s *workflowServer) DeleteWorkflow(ctx context.Context, req *workflowpkg.WorkflowDeleteRequest) (*workflowpkg.WorkflowDeleteResponse, error) {
	_, err := s.getWorkflow(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(req.Namespace).Delete(req.Name, &metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return &workflowpkg.WorkflowDeleteResponse{}, nil
}

func (s *workflowServer) RetryWorkflow(ctx context.Context, req *workflowpkg.WorkflowRetryRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)

	wf, err := s.getWorkflow(ctx, req.Namespace, req.Name, metav1.GetOptions{})
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
	wf, err := s.getWorkflow(ctx, req.Namespace, req.Name, metav1.GetOptions{})
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

	_, err := s.getWorkflow(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = util.ResumeWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), req.Name, req.NodeFieldSelector)
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

	_, err := s.getWorkflow(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = util.SuspendWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), req.Name)
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

	_, err := s.getWorkflow(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = util.TerminateWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), req.Name)
	if err != nil {
		return nil, err
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) StopWorkflow(ctx context.Context, req *workflowpkg.WorkflowStopRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	err := util.StopWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), req.Name, req.NodeFieldSelector, req.Message)
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

	_, err := validate.ValidateWorkflow(wftmplGetter, req.Workflow, validate.ValidateOpts{Lint: true})
	if err != nil {
		return nil, err
	}

	return req.Workflow, nil
}

func (s *workflowServer) PodLogs(req *workflowpkg.WorkflowLogRequest, ws workflowpkg.WorkflowService_PodLogsServer) error {
	ctx := ws.Context()
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)
	logger, err := logs.NewWorkflowLogger(wfClient, kubeClient, req, ws)
	if err != nil {
		return err
	}
	logger.Run(ctx)
	return nil
}

func (s *workflowServer) withInstanceID(opt metav1.ListOptions) metav1.ListOptions {
	if len(opt.LabelSelector) > 0 {
		opt.LabelSelector += ","
	}
	if len(s.instanceID) == 0 {
		opt.LabelSelector += fmt.Sprintf("!%s", common.LabelKeyControllerInstanceID)
		return opt
	}
	opt.LabelSelector += fmt.Sprintf("%s=%s", common.LabelKeyControllerInstanceID, s.instanceID)
	return opt
}

func (s *workflowServer) getWorkflow(ctx context.Context, namespace string, name string, options metav1.GetOptions) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(name, options)
	if err != nil {
		return nil, err
	}
	ok := s.validateInstanceID(wf)
	if !ok {
		return nil, fmt.Errorf("Workflow '%s' is not managed by the current Argo server", wf.Name)
	}
	return wf, nil
}

func (s *workflowServer) validateInstanceID(wf *v1alpha1.Workflow) bool {
	if len(s.instanceID) == 0 {
		if len(wf.Labels) == 0 {
			return true
		}
		if _, ok := wf.Labels[common.LabelKeyControllerInstanceID]; !ok {
			return true
		}
	} else if len(wf.Labels) > 0 {
		if val, ok := wf.Labels[common.LabelKeyControllerInstanceID]; ok {
			if val == s.instanceID {
				return true
			}
		}
	}
	return false
}
