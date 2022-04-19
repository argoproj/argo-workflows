package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	argoutil "github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/fields"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/logs"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
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
		return util.CreateServerDryRun(ctx, req.Workflow, wfClient)
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Create(ctx, req.Workflow, metav1.CreateOptions{})
	if err != nil {
		if apierr.IsServerTimeout(err) && req.Workflow.GenerateName != "" && req.Workflow.Name != "" {
			errWithHint := fmt.Errorf(`create request failed due to timeout, but it's possible that workflow "%s" already exists. Original error: %w`, req.Workflow.Name, err)
			log.Error(errWithHint)
			return nil, errWithHint
		}
		log.Errorf("Create request failed: %s", err)
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
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, wfGetOption)
	if err != nil {
		return nil, err
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}
	cleaner := fields.NewCleaner(req.Fields)
	if !cleaner.WillExclude("status.nodes") {
		if err := s.hydrator.Hydrate(wf); err != nil {
			return nil, err
		}
	}
	newWf := &wfv1.Workflow{}
	if ok, err := cleaner.Clean(wf, &newWf); err != nil {
		return nil, fmt.Errorf("unable to CleanFields in request: %w", err)
	} else if ok {
		return newWf, nil
	}
	return wf, nil
}

func (s *workflowServer) ListWorkflows(ctx context.Context, req *workflowpkg.WorkflowListRequest) (*wfv1.WorkflowList, error) {
	wfClient := auth.GetWfClient(ctx)

	listOption := &metav1.ListOptions{}
	if req.ListOptions != nil {
		listOption = req.ListOptions
	}
	s.instanceIDService.With(listOption)
	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).List(ctx, *listOption)
	if err != nil {
		return nil, err
	}
	cleaner := fields.NewCleaner(req.Fields)
	if s.offloadNodeStatusRepo.IsEnabled() && !cleaner.WillExclude("items.status.nodes") {
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

	res := &wfv1.WorkflowList{ListMeta: metav1.ListMeta{Continue: wfList.Continue, ResourceVersion: wfList.ResourceVersion}, Items: wfList.Items}
	newRes := &wfv1.WorkflowList{}
	if ok, err := cleaner.Clean(res, &newRes); err != nil {
		return nil, fmt.Errorf("unable to CleanFields in request: %w", err)
	} else if ok {
		return newRes, nil
	}
	return res, nil
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
			wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, wfName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			opts.FieldSelector = argoutil.GenerateFieldSelectorFromWorkflowName(wf.Name)
		}
	}
	s.instanceIDService.With(opts)
	wfIf := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace)
	watch, err := wfIf.Watch(ctx, *opts)
	if err != nil {
		return err
	}
	defer watch.Stop()
	cleaner := fields.NewCleaner(req.Fields).WithoutPrefix("result.object.")

	clean := func(x *wfv1.Workflow) (*wfv1.Workflow, error) {
		y := &wfv1.Workflow{}
		if clean, err := cleaner.Clean(x, y); err != nil {
			return nil, err
		} else if clean {
			return y, nil
		} else {
			return x, nil
		}
	}
	log.Debug("Piping events to channel")
	defer log.Debug("Result channel done")

	// Eagerly send the headers so that we can begin our keepalive loop if no results are received
	// immediately.  Without this, we cannot detect a streaming response, and we can't write to the
	// response since a subsequent write by the stream causes an error.
	err = ws.SendHeader(metadata.MD{})

	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, open := <-watch.ResultChan():
			if !open {
				return io.EOF
			}
			log.Debug("Received workflow event")
			wf, ok := event.Object.(*wfv1.Workflow)
			if !ok {
				// object is probably metav1.Status, `FromObject` can deal with anything
				return apierr.FromObject(event.Object)
			}
			logCtx := log.WithFields(log.Fields{"workflow": wf.Name, "type": event.Type, "phase": wf.Status.Phase})
			if !cleaner.WillExclude("status.nodes") {
				if err := s.hydrator.Hydrate(wf); err != nil {
					return err
				}
			}
			newWf, err := clean(wf)
			if err != nil {
				return fmt.Errorf("unable to CleanFields in request: %w", err)
			}
			logCtx.Debug("Sending workflow event")
			err = ws.Send(&workflowpkg.WorkflowWatchEvent{Type: string(event.Type), Object: newWf})
			if err != nil {
				return err
			}
		}
	}
}

func (s *workflowServer) WatchEvents(req *workflowpkg.WatchEventsRequest, ws workflowpkg.WorkflowService_WatchEventsServer) error {
	ctx := ws.Context()
	kubeClient := auth.GetKubeClient(ctx)
	opts := &metav1.ListOptions{}
	if req.ListOptions != nil {
		opts = req.ListOptions
	}
	s.instanceIDService.With(opts)
	eventInterface := kubeClient.CoreV1().Events(req.Namespace)
	watch, err := eventInterface.Watch(ctx, *opts)
	if err != nil {
		return err
	}
	defer watch.Stop()

	log.Debug("Piping events to channel")
	defer log.Debug("Result channel done")

	err = ws.SendHeader(metadata.MD{})

	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, open := <-watch.ResultChan():
			if !open {
				return io.EOF
			}
			log.Debug("Received event")
			e, ok := event.Object.(*corev1.Event)
			if !ok {
				// object is probably probably metav1.Status, `FromObject` can deal with anything
				return apierr.FromObject(event.Object)
			}
			log.Debug("Sending event")
			err = ws.Send(e)
			if err != nil {
				return err
			}
		}
	}
}

func (s *workflowServer) DeleteWorkflow(ctx context.Context, req *workflowpkg.WorkflowDeleteRequest) (*workflowpkg.WorkflowDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}
	err = auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(wf.Namespace).Delete(ctx, wf.Name, metav1.DeleteOptions{PropagationPolicy: argoutil.GetDeletePropagation()})
	if err != nil {
		return nil, err
	}
	return &workflowpkg.WorkflowDeleteResponse{}, nil
}

func (s *workflowServer) RetryWorkflow(ctx context.Context, req *workflowpkg.WorkflowRetryRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)

	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
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

	wf, podsToDelete, err := util.FormulateRetryWorkflow(ctx, wf, req.RestartSuccessful, req.NodeFieldSelector)
	if err != nil {
		return nil, err
	}

	for _, podName := range podsToDelete {
		log.WithFields(log.Fields{"podDeleted": podName}).Info("Deleting pod")
		err := kubeClient.CoreV1().Pods(wf.Namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		if err != nil && !apierr.IsNotFound(err) {
			return nil, err
		}
	}

	err = s.hydrator.Dehydrate(wf)
	if err != nil {
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Update(ctx, wf, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *workflowServer) ResubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowResubmitRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
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

	created, err := util.SubmitWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wfClient, req.Namespace, newWF, &wfv1.SubmitOpts{})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *workflowServer) ResumeWorkflow(ctx context.Context, req *workflowpkg.WorkflowResumeRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}

	err = util.ResumeWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, wf.Name, req.NodeFieldSelector)
	if err != nil {
		log.Warnf("Failed to resume %s: %+v", wf.Name, err)
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *workflowServer) SuspendWorkflow(ctx context.Context, req *workflowpkg.WorkflowSuspendRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}

	err = util.SuspendWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(wf.Namespace), wf.Name)
	if err != nil {
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wf, nil
}

func (s *workflowServer) TerminateWorkflow(ctx context.Context, req *workflowpkg.WorkflowTerminateRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}

	err = util.TerminateWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wf.Name)
	if err != nil {
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) StopWorkflow(ctx context.Context, req *workflowpkg.WorkflowStopRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, err
	}
	err = util.StopWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, wf.Name, req.NodeFieldSelector, req.Message)
	if err != nil {
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) SetWorkflow(ctx context.Context, req *workflowpkg.WorkflowSetRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
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

	err = util.SetWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, wf.Name, req.NodeFieldSelector, operation)
	if err != nil {
		return nil, err
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) LintWorkflow(ctx context.Context, req *workflowpkg.WorkflowLintRequest) (*wfv1.Workflow, error) {
	if req.Workflow == nil {
		return nil, fmt.Errorf("unable to get a workflow")
	}
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
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return err
	}
	req.Name = wf.Name

	err = ws.SendHeader(metadata.MD{})

	if err != nil {
		return err
	}

	return logs.WorkflowLogs(ctx, wfClient, kubeClient, req, ws)
}

func (s *workflowServer) WorkflowLogs(req *workflowpkg.WorkflowLogRequest, ws workflowpkg.WorkflowService_WorkflowLogsServer) error {
	return s.PodLogs(req, ws)
}

func (s *workflowServer) getWorkflow(ctx context.Context, wfClient versioned.Interface, namespace string, name string, options metav1.GetOptions) (*wfv1.Workflow, error) {
	if name == latestAlias {
		latest, err := getLatestWorkflow(ctx, wfClient, namespace)
		if err != nil {
			return nil, err
		}
		log.Debugf("Resolved alias %s to workflow %s.\n", latestAlias, latest.Name)
		return latest, nil
	}
	wf, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, name, options)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (s *workflowServer) validateWorkflow(wf *wfv1.Workflow) error {
	return s.instanceIDService.Validate(wf)
}

func getLatestWorkflow(ctx context.Context, wfClient versioned.Interface, namespace string) (*wfv1.Workflow, error) {
	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(ctx, metav1.ListOptions{})
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
		cronWf, err := wfClient.ArgoprojV1alpha1().CronWorkflows(req.Namespace).Get(ctx, req.ResourceName, metav1.GetOptions{})
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
	creator.Label(ctx, wf)
	err := util.ApplySubmitOpts(wf, req.SubmitOptions)
	if err != nil {
		return nil, err
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	_, err = validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, validate.ValidateOpts{Submit: true})
	if err != nil {
		return nil, err
	}
	return wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Create(ctx, wf, metav1.CreateOptions{})
}
