package workflow

import (
	"encoding/json"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/persist/sqldb"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/server/auth"
	argoutil "github.com/argoproj/argo/util"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/util/logs"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/creator"
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

const latestAlias = "@latest"

// NewWorkflowServer returns a new workflowServer
func NewWorkflowServer(instanceIDService instanceid.Service, offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo) workflowpkg.WorkflowServiceServer {
	return &workflowServer{instanceIDService, offloadNodeStatusRepo, hydrator.New(offloadNodeStatusRepo)}
}

func (s *workflowServer) CreateWorkflow(ctx context.Context, req *workflowpkg.WorkflowCreateRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	if req.Workflow == nil {
		return nil, fmt.Errorf("workflow body not specified")
	}

	if req.Workflow.Namespace == "" {
		req.Workflow.Namespace = req.Namespace
	}

	s.instanceIDService.Label(req.Workflow)
	creator.Label(ctx, req.Workflow)

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

func (s *workflowServer) GetWorkflow(ctx context.Context, req *workflowpkg.WorkflowGetRequest) (*wfv1.Workflow, error) {
	wfGetOption := metav1.GetOptions{}
	if req.GetOptions != nil {
		wfGetOption = *req.GetOptions
	}
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(wfClient, req.Namespace, req.Name, wfGetOption)
	if err != nil {
		return nil, err
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}
	err = s.hydrator.Hydrate(wf)
	if err != nil {
		return nil, err
	}
	return wf, err
}

func (s *workflowServer) ListWorkflows(ctx context.Context, req *workflowpkg.WorkflowListRequest) (*wfv1.WorkflowList, error) {
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

	return &wfv1.WorkflowList{ListMeta: metav1.ListMeta{Continue: wfList.Continue, ResourceVersion: wfList.ResourceVersion}, Items: wfList.Items}, nil
}

func (s *workflowServer) WatchWorkflows(req *workflowpkg.WatchWorkflowsRequest, ws workflowpkg.WorkflowService_WatchWorkflowsServer) error {
	ctx := ws.Context()
	wfClient := auth.GetWfClient(ctx)
	opts := &metav1.ListOptions{}
	if req.ListOptions != nil {
		opts = req.ListOptions
		wfName := argoutil.RecoverWorkflowNameFromSelectorStringIfAny(opts.FieldSelector)
		if wfName != "" {
			// If we are using an alias (such as `@latest`) we need to dereference it.
			// s.getWorkflow does that for us
			wf, err := s.getWorkflow(wfClient, req.Namespace, wfName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			opts.FieldSelector = argoutil.GenerateFieldSelectorFromWorkflowName(wf.Name)
		}
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
			return nil
		case event, open := <-watch.ResultChan():
			if !open {
				log.Debug("Re-establishing workflow watch")
				watch.Stop()
				watch, err = wfIf.Watch(*opts)
				if err != nil {
					return err
				}
				continue
			}
			log.Debug("Received event")
			wf, ok := event.Object.(*wfv1.Workflow)
			if !ok {
				// object is probably probably metav1.Status, `FromObject` can deal with anything
				return apierr.FromObject(event.Object)
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
			// when we re-establish, we want to start at the same place
			opts.ResourceVersion = wf.ResourceVersion
		}
	}
}

func (s *workflowServer) DeleteWorkflow(ctx context.Context, req *workflowpkg.WorkflowDeleteRequest) (*workflowpkg.WorkflowDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}
	err = auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(wf.Namespace).Delete(wf.Name, &metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return &workflowpkg.WorkflowDeleteResponse{}, nil
}

func (s *workflowServer) RetryWorkflow(ctx context.Context, req *workflowpkg.WorkflowRetryRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)

	wf, err := s.getWorkflow(wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}

	wf, err = util.RetryWorkflow(kubeClient, s.hydrator, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wf, req.RestartSuccessful, req.NodeFieldSelector)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) ResubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowResubmitRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}

	newWF, err := util.FormulateResubmitWorkflow(wf, req.Memoized)
	if err != nil {
		return nil, err
	}

	created, err := util.SubmitWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wfClient, req.Namespace, newWF, &wfv1.SubmitOpts{})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *workflowServer) ResumeWorkflow(ctx context.Context, req *workflowpkg.WorkflowResumeRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}

	err = util.ResumeWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, wf.Name, req.NodeFieldSelector)
	if err != nil {
		log.Warnf("Failed to resume %s: %+v", wf.Name, err)
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *workflowServer) SuspendWorkflow(ctx context.Context, req *workflowpkg.WorkflowSuspendRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wf, err := s.getWorkflow(wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}

	err = util.SuspendWorkflow(wfClient.ArgoprojV1alpha1().Workflows(wf.Namespace), wf.Name)
	if err != nil {
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *workflowServer) TerminateWorkflow(ctx context.Context, req *workflowpkg.WorkflowTerminateRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wf, err := s.getWorkflow(wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}

	err = util.TerminateWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wf.Name)
	if err != nil {
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) StopWorkflow(ctx context.Context, req *workflowpkg.WorkflowStopRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}
	err = util.StopWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, wf.Name, req.NodeFieldSelector, req.Message)
	if err != nil {
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) SetWorkflow(ctx context.Context, req *workflowpkg.WorkflowSetRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}

	phaseToSet := wfv1.NodePhase(req.Phase)
	switch phaseToSet {
	case wfv1.NodeSucceeded, wfv1.NodeFailed, wfv1.NodeError, "":
		// Do nothing, passes validation
	default:
		return nil, fmt.Errorf("%s is an invalid phase to set to", req.Phase)
	}

	outputParams := make(map[string]string)
	if req.OutputParameters != "" {
		err = json.Unmarshal([]byte(req.OutputParameters), &outputParams)
		if err != nil {
			return nil, fmt.Errorf("unable to parse output parameter set request: %s", err)
		}
	}

	operation := util.SetOperationValues{
		Phase:            phaseToSet,
		Message:          req.Message,
		OutputParameters: outputParams,
	}

	err = util.SetWorkflow(wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, wf.Name, req.NodeFieldSelector, operation)
	if err != nil {
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) LintWorkflow(ctx context.Context, req *workflowpkg.WorkflowLintRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	s.instanceIDService.Label(req.Workflow)
	creator.Label(ctx, req.Workflow)

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
	wf, err := s.getWorkflow(wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return err
	}
	req.Name = wf.Name
	return logs.WorkflowLogs(ctx, wfClient, kubeClient, req, ws)
}

func (s *workflowServer) getWorkflow(wfClient versioned.Interface, namespace string, name string, options metav1.GetOptions) (*wfv1.Workflow, error) {
	if name == latestAlias {
		latest, err := getLatestWorkflow(wfClient, namespace)
		if err != nil {
			return nil, err
		}
		log.Debugf("Resolved alias %s to workflow %s.\n", latestAlias, latest.Name)
		return latest, nil
	}
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(name, options)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) validateWorkflow(wf *wfv1.Workflow) error {
	return s.instanceIDService.Validate(wf)
}

func getLatestWorkflow(wfClient versioned.Interface, namespace string) (*wfv1.Workflow, error) {
	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	if len(wfList.Items) < 1 {
		return nil, fmt.Errorf("No workflows found.")
	}
	latest := wfList.Items[0]
	for _, wf := range wfList.Items {
		if latest.ObjectMeta.CreationTimestamp.Before(&wf.ObjectMeta.CreationTimestamp) {
			latest = wf
		}
	}
	return &latest, nil
}

func (s *workflowServer) SubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowSubmitRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	var wf *wfv1.Workflow
	switch req.ResourceKind {
	case workflow.CronWorkflowKind, workflow.CronWorkflowSingular, workflow.CronWorkflowPlural, workflow.CronWorkflowShortName:
		cronWf, err := wfClient.ArgoprojV1alpha1().CronWorkflows(req.Namespace).Get(req.ResourceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		wf = common.ConvertCronWorkflowToWorkflow(cronWf)
	case workflow.WorkflowTemplateKind, workflow.WorkflowTemplateSingular, workflow.WorkflowTemplatePlural, workflow.WorkflowTemplateShortName:
		wfTmpl, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Get(req.ResourceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		wf = common.NewWorkflowFromWorkflowTemplate(req.ResourceName, wfTmpl.Spec.WorkflowMetadata, false)
	case workflow.ClusterWorkflowTemplateKind, workflow.ClusterWorkflowTemplateSingular, workflow.ClusterWorkflowTemplatePlural, workflow.ClusterWorkflowTemplateShortName:
		cwfTmpl, err := wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().Get(req.ResourceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		wf = common.NewWorkflowFromWorkflowTemplate(req.ResourceName, cwfTmpl.Spec.WorkflowMetadata, true)
	default:
		return nil, errors.Errorf(errors.CodeBadRequest, "Resource kind '%s' is not supported for submitting", req.ResourceKind)
	}

	s.instanceIDService.Label(wf)
	creator.Label(ctx, wf)
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
