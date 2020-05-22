package workflow

import (
	"fmt"
	"reflect"
	"sort"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/persist/sqldb"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/util/logs"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/util"
	"github.com/argoproj/argo/workflow/validate"
)

type workflowServer struct {
	instanceIDService     instanceid.Service
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	hydrator              hydrator.Interface
}

// NewWorkflowServer returns a new workflowServer
func NewWorkflowServer(instanceIDService instanceid.Service, offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo) workflowpkg.WorkflowServiceServer {
	return &workflowServer{instanceIDService, offloadNodeStatusRepo, hydrator.New(offloadNodeStatusRepo)}
}

func (s *workflowServer) CreateWorkflow(ctx context.Context, req *workflowpkg.WorkflowCreateRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	if req.Workflow == nil {
		return nil, fmt.Errorf("workflow body not specified")
	}

	if req.Workflow.Namespace == "" {
		req.Workflow.Namespace = req.Namespace
	}

	s.instanceIDService.Label(req.Workflow)

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	_, err := validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, req.Workflow, validate.ValidateOpts{})

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
	wf, err := s.getWorkflowAndValidate(ctx, req.Namespace, req.Name, wfGetOption)
	if err != nil {
		return nil, err
	}
	err = s.hydrator.Hydrate(wf)
	if err != nil {
		return nil, err
	}
	return wf, err
}

func (s *workflowServer) ListWorkflows(ctx context.Context, req *workflowpkg.WorkflowListRequest) (*v1alpha1.WorkflowList, error) {
	wfClient := auth.GetWfClient(ctx)

	var listOption = &metav1.ListOptions{}
	if req.ListOptions != nil {
		listOption = req.ListOptions
	}
	s.instanceIDService.With(listOption)
	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).List(*listOption)
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
					log.WithFields(log.Fields{"namespace": wf.Namespace, "name": wf.Name}).Warn(sqldb.OffloadNodeStatusDisabled)
				}
			}
		}
	}

	// we make no promises about the overall list sorting, we just sort each page
	sort.Sort(wfList.Items)

	return &v1alpha1.WorkflowList{ListMeta: metav1.ListMeta{Continue: wfList.Continue}, Items: wfList.Items}, nil
}

func (s *workflowServer) WatchWorkflows(req *workflowpkg.WatchWorkflowsRequest, ws workflowpkg.WorkflowService_WatchWorkflowsServer) error {
	ctx := ws.Context()
	wfClient := auth.GetWfClient(ctx)
	opts := &metav1.ListOptions{}
	if req.ListOptions != nil {
		opts = req.ListOptions
	}
	s.instanceIDService.With(opts)
	wfIf := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace)
	watch, err := wfIf.Watch(*opts)
	if err != nil {
		return err
	}
	defer watch.Stop()

	log.Debug("Piping events to channel")
	defer log.Debug("Result channel done")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, open := <-watch.ResultChan():
			if !open {
				log.Debug("Re-establishing workflow watch")
				watch, err = wfIf.Watch(*opts)
				if err != nil {
					return err
				}
				continue
			}
			log.Debug("Received event")
			wf, ok := event.Object.(*v1alpha1.Workflow)
			if !ok {
				return fmt.Errorf("watch object was not a workflow %v", reflect.TypeOf(event.Object))
			}
			logCtx := log.WithFields(log.Fields{"workflow": wf.Name, "type": event.Type, "phase": wf.Status.Phase})
			err := s.hydrator.Hydrate(wf)
			if err != nil {
				return err
			}
			logCtx.Debug("Sending event")
			err = ws.Send(&workflowpkg.WorkflowWatchEvent{Type: string(event.Type), Object: wf})
			if err != nil {
				return err
			}
		}
	}
}

func (s *workflowServer) DeleteWorkflow(ctx context.Context, req *workflowpkg.WorkflowDeleteRequest) (*workflowpkg.WorkflowDeleteResponse, error) {
	_, err := s.getWorkflowAndValidate(ctx, req.Namespace, req.Name, metav1.GetOptions{})
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

	wf, err := s.getWorkflowAndValidate(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	wf, err = util.RetryWorkflow(kubeClient, s.hydrator, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wf, req.RestartSuccessful, req.NodeFieldSelector)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) ResubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowResubmitRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflowAndValidate(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	newWF, err := util.FormulateResubmitWorkflow(wf, req.Memoized)
	if err != nil {
		return nil, err
	}

	created, err := util.SubmitWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wfClient, req.Namespace, newWF, &v1alpha1.SubmitOpts{})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *workflowServer) ResumeWorkflow(ctx context.Context, req *workflowpkg.WorkflowResumeRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	_, err := s.getWorkflowAndValidate(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = util.ResumeWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, req.Name, req.NodeFieldSelector)
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

	_, err := s.getWorkflowAndValidate(ctx, req.Namespace, req.Name, metav1.GetOptions{})
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

	_, err := s.getWorkflowAndValidate(ctx, req.Namespace, req.Name, metav1.GetOptions{})
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
	_, err := s.getWorkflowAndValidate(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = util.StopWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, req.Name, req.NodeFieldSelector, req.Message)
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
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	s.instanceIDService.Label(req.Workflow)

	_, err := validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, req.Workflow, validate.ValidateOpts{Lint: true})

	if err != nil {
		return nil, err
	}

	return req.Workflow, nil
}

func (s *workflowServer) PodLogs(req *workflowpkg.WorkflowLogRequest, ws workflowpkg.WorkflowService_PodLogsServer) error {
	ctx := ws.Context()
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)
	_, err := s.getWorkflowAndValidate(ctx, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return logs.WorkflowLogs(ctx, wfClient, kubeClient, req, ws)
}

func (s *workflowServer) getWorkflowAndValidate(ctx context.Context, namespace string, name string, options metav1.GetOptions) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(name, options)
	if err != nil {
		return nil, err
	}
	err = s.instanceIDService.Validate(wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) SubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowSubmitRequest) (*v1alpha1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	var wf *v1alpha1.Workflow
	switch req.ResourceKind {
	case workflow.CronWorkflowKind, workflow.CronWorkflowSingular, workflow.CronWorkflowPlural, workflow.CronWorkflowShortName:
		cronWf, err := wfClient.ArgoprojV1alpha1().CronWorkflows(req.Namespace).Get(req.ResourceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		wf = common.ConvertCronWorkflowToWorkflow(cronWf)
	case workflow.WorkflowTemplateKind, workflow.WorkflowTemplateSingular, workflow.WorkflowTemplatePlural, workflow.WorkflowTemplateShortName:
		wf = common.NewWorkflowFromWorkflowTemplate(req.ResourceName, false)
	case workflow.ClusterWorkflowTemplateKind, workflow.ClusterWorkflowTemplateSingular, workflow.ClusterWorkflowTemplatePlural, workflow.ClusterWorkflowTemplateShortName:
		wf = common.NewWorkflowFromWorkflowTemplate(req.ResourceName, true)
	default:
		return nil, errors.Errorf(errors.CodeBadRequest, "Resource kind '%s' is not supported for submitting", req.ResourceKind)
	}

	s.instanceIDService.Label(wf)
	err := util.ApplySubmitOpts(wf, req.SubmitOptions)
	if err != nil {
		return nil, err
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	_, err = validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, validate.ValidateOpts{})
	if err != nil {
		return nil, err
	}
	return wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Create(wf)

}
