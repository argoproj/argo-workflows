package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"sync"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	corev1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
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
	wfArchiveServer       workflowarchivepkg.ArchivedWorkflowServiceServer
}

const latestAlias = "@latest"

// NewWorkflowServer returns a new workflowServer
func NewWorkflowServer(instanceIDService instanceid.Service, offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo, wfArchiveServer workflowarchivepkg.ArchivedWorkflowServiceServer) workflowpkg.WorkflowServiceServer {
	return &workflowServer{instanceIDService, offloadNodeStatusRepo, hydrator.New(offloadNodeStatusRepo), wfArchiveServer}
}

func (s *workflowServer) CreateWorkflow(ctx context.Context, req *workflowpkg.WorkflowCreateRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	if req.Workflow == nil {
		return nil, sutils.ToStatusError(fmt.Errorf("workflow body not specified"), codes.InvalidArgument)
	}

	if req.Workflow.Namespace == "" {
		req.Workflow.Namespace = req.Namespace
	}

	s.instanceIDService.Label(req.Workflow)
	creator.Label(ctx, req.Workflow)

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	err := validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, req.Workflow, validate.ValidateOpts{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}

	// if we are doing a normal dryRun, just return the workflow un-altered
	if req.CreateOptions != nil && len(req.CreateOptions.DryRun) > 0 {
		return req.Workflow, nil
	}
	if req.ServerDryRun {
		workflow, err := util.CreateServerDryRun(ctx, req.Workflow, wfClient)
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.InvalidArgument)
		}
		return workflow, nil
	}

	wf, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Create(ctx, req.Workflow, metav1.CreateOptions{})
	if err != nil {
		if apierr.IsServerTimeout(err) && req.Workflow.GenerateName != "" && req.Workflow.Name != "" {
			errWithHint := fmt.Errorf(`create request failed due to timeout, but it's possible that workflow "%s" already exists. Original error: %w`, req.Workflow.Name, err)
			log.WithError(errWithHint).Error(errWithHint.Error())
			return nil, sutils.ToStatusError(errWithHint, codes.DeadlineExceeded)
		}
		log.WithError(err).Error("Create request failed")
		return nil, sutils.ToStatusError(err, codes.Internal)
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
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	cleaner := fields.NewCleaner(req.Fields)
	if !cleaner.WillExclude("status.nodes") {
		if err := s.hydrator.Hydrate(wf); err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
	}
	newWf := &wfv1.Workflow{}
	if ok, err := cleaner.Clean(wf, &newWf); err != nil {
		// should this be InvalidArgument?
		return nil, sutils.ToStatusError(fmt.Errorf("unable to CleanFields in request: %w", err), codes.Internal)
	} else if ok {
		return newWf, nil
	}
	return wf, nil
}

func mergeWithArchivedWorkflows(liveWfs wfv1.WorkflowList, archivedWfs wfv1.WorkflowList, numWfsToKeep int) *wfv1.WorkflowList {
	var mergedWfs []wfv1.Workflow
	var uidToWfs = map[types.UID][]wfv1.Workflow{}
	for _, item := range liveWfs.Items {
		uidToWfs[item.UID] = append(uidToWfs[item.UID], item)
	}
	for _, item := range archivedWfs.Items {
		uidToWfs[item.UID] = append(uidToWfs[item.UID], item)
	}

	for _, v := range uidToWfs {
		// The archived workflow we saved in the database have "Persisted" as the archival status.
		// Prioritize 'Archived' over 'Persisted' because 'Archived' means the workflow is in the cluster
		if len(v) == 1 {
			mergedWfs = append(mergedWfs, v[0])
		} else {
			if ok := v[0].Labels[common.LabelKeyWorkflowArchivingStatus] == "Archived"; ok {
				mergedWfs = append(mergedWfs, v[0])
			} else {
				mergedWfs = append(mergedWfs, v[1])
			}
		}
	}
	mergedWfsList := wfv1.WorkflowList{Items: mergedWfs, ListMeta: liveWfs.ListMeta}
	sort.Sort(mergedWfsList.Items)
	numWfs := 0
	var finalWfs []wfv1.Workflow
	for _, item := range mergedWfsList.Items {
		if numWfsToKeep == 0 || numWfs < numWfsToKeep {
			finalWfs = append(finalWfs, item)
			numWfs += 1
		}
	}
	return &wfv1.WorkflowList{Items: finalWfs, ListMeta: liveWfs.ListMeta}
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
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	archivedWfList, err := s.wfArchiveServer.ListArchivedWorkflows(ctx, &workflowarchivepkg.ListArchivedWorkflowsRequest{
		ListOptions: listOption,
		NamePrefix:  "",
		Namespace:   req.Namespace,
	})
	if err != nil {
		log.Warnf("unable to list archived workflows:%v", err)
	} else {
		if archivedWfList != nil {
			wfList = mergeWithArchivedWorkflows(*wfList, *archivedWfList, int(listOption.Limit))
		}
	}

	cleaner := fields.NewCleaner(req.Fields)
	if s.offloadNodeStatusRepo.IsEnabled() && !cleaner.WillExclude("items.status.nodes") {
		offloadedNodes, err := s.offloadNodeStatusRepo.List(req.Namespace)
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
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
		return nil, sutils.ToStatusError(fmt.Errorf("unable to CleanFields in request: %w", err), codes.Internal)
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
				return sutils.ToStatusError(err, codes.Internal)
			}
			opts.FieldSelector = argoutil.GenerateFieldSelectorFromWorkflowName(wf.Name)
		}
	}
	s.instanceIDService.With(opts)
	wfIf := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace)
	watch, err := wfIf.Watch(ctx, *opts)
	if err != nil {
		return sutils.ToStatusError(err, codes.Internal)
	}
	defer watch.Stop()
	cleaner := fields.NewCleaner(req.Fields).WithoutPrefix("result.object.")

	clean := func(x *wfv1.Workflow) (*wfv1.Workflow, error) {
		y := &wfv1.Workflow{}
		if clean, err := cleaner.Clean(x, y); err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
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
				return sutils.ToStatusError(io.EOF, codes.ResourceExhausted)
			}
			log.Debug("Received workflow event")
			wf, ok := event.Object.(*wfv1.Workflow)
			if !ok {
				// object is probably metav1.Status, `FromObject` can deal with anything
				return sutils.ToStatusError(apierr.FromObject(event.Object), codes.Internal)
			}
			logCtx := log.WithFields(log.Fields{"workflow": wf.Name, "type": event.Type, "phase": wf.Status.Phase})
			if !cleaner.WillExclude("status.nodes") {
				if err := s.hydrator.Hydrate(wf); err != nil {
					return sutils.ToStatusError(err, codes.Internal)
				}
			}
			newWf, err := clean(wf)
			if err != nil {
				return sutils.ToStatusError(fmt.Errorf("unable to CleanFields in request: %w", err), codes.Internal)
			}
			logCtx.Debug("Sending workflow event")
			err = ws.Send(&workflowpkg.WorkflowWatchEvent{Type: string(event.Type), Object: newWf})
			if err != nil {
				return sutils.ToStatusError(err, codes.Internal)
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
		return sutils.ToStatusError(err, codes.Internal)
	}
	defer watch.Stop()

	log.Debug("Piping events to channel")
	defer log.Debug("Result channel done")

	err = ws.SendHeader(metadata.MD{})

	if err != nil {
		return sutils.ToStatusError(err, codes.Internal)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, open := <-watch.ResultChan():
			if !open {
				return sutils.ToStatusError(io.EOF, codes.ResourceExhausted)
			}
			log.Debug("Received event")
			e, ok := event.Object.(*corev1.Event)
			if !ok {
				// object is probably probably metav1.Status, `FromObject` can deal with anything
				return sutils.ToStatusError(apierr.FromObject(event.Object), codes.Internal)
			}
			log.Debug("Sending event")
			err = ws.Send(e)
			if err != nil {
				return sutils.ToStatusError(err, codes.Internal)
			}
		}
	}
}

func (s *workflowServer) DeleteWorkflow(ctx context.Context, req *workflowpkg.WorkflowDeleteRequest) (*workflowpkg.WorkflowDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	if req.Force {
		_, err := auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(wf.Namespace).Patch(ctx, wf.Name, types.MergePatchType, []byte("{\"metadata\":{\"finalizers\":null}}"), metav1.PatchOptions{})
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
	}
	err = auth.GetWfClient(ctx).ArgoprojV1alpha1().Workflows(wf.Namespace).Delete(ctx, wf.Name, metav1.DeleteOptions{PropagationPolicy: argoutil.GetDeletePropagation()})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return &workflowpkg.WorkflowDeleteResponse{}, nil
}

func errorFromChannel(errCh <-chan error) error {
	select {
	case err := <-errCh:
		return err
	default:
	}
	return nil
}

func (s *workflowServer) RetryWorkflow(ctx context.Context, req *workflowpkg.WorkflowRetryRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)

	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}

	err = s.hydrator.Hydrate(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	wf, podsToDelete, err := util.FormulateRetryWorkflow(ctx, wf, req.RestartSuccessful, req.NodeFieldSelector, req.Parameters)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	errCh := make(chan error, len(podsToDelete))
	var wg sync.WaitGroup
	wg.Add(len(podsToDelete))
	for _, podName := range podsToDelete {
		log.WithFields(log.Fields{"podDeleted": podName}).Info("Deleting pod")
		go func(podName string) {
			defer wg.Done()
			err := kubeClient.CoreV1().Pods(wf.Namespace).Delete(ctx, podName, metav1.DeleteOptions{})
			if err != nil && !apierr.IsNotFound(err) {
				errCh <- err
				return
			}
		}(podName)
	}
	wg.Wait()

	err = errorFromChannel(errCh)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	err = s.hydrator.Dehydrate(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Update(ctx, wf, metav1.UpdateOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	return wf, nil
}

func (s *workflowServer) ResubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowResubmitRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}

	newWF, err := util.FormulateResubmitWorkflow(ctx, wf, req.Memoized, req.Parameters)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	created, err := util.SubmitWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wfClient, req.Namespace, newWF, &wfv1.SubmitOpts{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return created, nil
}

func (s *workflowServer) ResumeWorkflow(ctx context.Context, req *workflowpkg.WorkflowResumeRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}

	err = util.ResumeWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, wf.Name, req.NodeFieldSelector)
	if err != nil {
		log.WithFields(log.Fields{"name": wf.Name}).WithError(err).Warn("Failed to resume")
		return nil, sutils.ToStatusError(err, codes.Internal)

	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	return wf, nil
}

func (s *workflowServer) SuspendWorkflow(ctx context.Context, req *workflowpkg.WorkflowSuspendRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}

	err = util.SuspendWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(wf.Namespace), wf.Name)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	return wf, nil
}

func (s *workflowServer) TerminateWorkflow(ctx context.Context, req *workflowpkg.WorkflowTerminateRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}

	err = util.TerminateWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wf.Name)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return wf, nil
}

func (s *workflowServer) StopWorkflow(ctx context.Context, req *workflowpkg.WorkflowStopRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	err = util.StopWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, wf.Name, req.NodeFieldSelector, req.Message)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return wf, nil
}

func (s *workflowServer) SetWorkflow(ctx context.Context, req *workflowpkg.WorkflowSetRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	wf, err := s.getWorkflow(ctx, wfClient, req.Namespace, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}

	phaseToSet := wfv1.NodePhase(req.Phase)
	switch phaseToSet {
	case wfv1.NodeSucceeded, wfv1.NodeFailed, wfv1.NodeError, "":
		// Do nothing, passes validation
	default:
		return nil, sutils.ToStatusError(fmt.Errorf("%s is an invalid phase to set to", req.Phase), codes.InvalidArgument)
	}

	outputParams := make(map[string]string)
	if req.OutputParameters != "" {
		err = json.Unmarshal([]byte(req.OutputParameters), &outputParams)
		if err != nil {
			return nil, sutils.ToStatusError(fmt.Errorf("unable to parse output parameter set request: %s", err), codes.InvalidArgument)
		}
	}

	operation := util.SetOperationValues{
		Phase:            phaseToSet,
		Message:          req.Message,
		OutputParameters: outputParams,
	}

	err = util.SetWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), s.hydrator, wf.Name, req.NodeFieldSelector, operation)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
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

	err := validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, req.Workflow, validate.ValidateOpts{Lint: true})
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
		return sutils.ToStatusError(err, codes.Internal)
	}
	err = s.validateWorkflow(wf)
	if err != nil {
		return sutils.ToStatusError(err, codes.InvalidArgument)
	}
	req.Name = wf.Name

	err = ws.SendHeader(metadata.MD{})

	if err != nil {
		return sutils.ToStatusError(err, codes.Internal)
	}

	err = logs.WorkflowLogs(ctx, wfClient, kubeClient, req, ws)
	return sutils.ToStatusError(err, codes.Internal)
}

func (s *workflowServer) WorkflowLogs(req *workflowpkg.WorkflowLogRequest, ws workflowpkg.WorkflowService_WorkflowLogsServer) error {
	return sutils.ToStatusError(s.PodLogs(req, ws), codes.Internal)
}

func (s *workflowServer) getWorkflow(ctx context.Context, wfClient versioned.Interface, namespace string, name string, options metav1.GetOptions) (*wfv1.Workflow, error) {
	if name == latestAlias {
		latest, err := getLatestWorkflow(ctx, wfClient, namespace)
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		log.Debugf("Resolved alias %s to workflow %s.\n", latestAlias, latest.Name)
		return latest, nil
	}
	var err error
	wf, origErr := wfClient.ArgoprojV1alpha1().Workflows(namespace).Get(ctx, name, options)
	if wf == nil || origErr != nil {
		wf, err = s.wfArchiveServer.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{
			Namespace: namespace,
			Name:      name,
		})
		if err != nil {
			log.Errorf("failed to get live workflow: %v; failed to get archived workflow: %v", origErr, err)
			// We only return the original error to preserve the original status code.
			return nil, sutils.ToStatusError(origErr, codes.Internal)
		}
	}
	return wf, nil
}

func (s *workflowServer) validateWorkflow(wf *wfv1.Workflow) error {
	return sutils.ToStatusError(s.instanceIDService.Validate(wf), codes.InvalidArgument)
}

func getLatestWorkflow(ctx context.Context, wfClient versioned.Interface, namespace string) (*wfv1.Workflow, error) {
	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if len(wfList.Items) < 1 {
		return nil, sutils.ToStatusError(fmt.Errorf("No workflows found."), codes.NotFound)
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
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		wf = common.ConvertCronWorkflowToWorkflow(cronWf)
	case workflow.WorkflowTemplateKind, workflow.WorkflowTemplateSingular, workflow.WorkflowTemplatePlural, workflow.WorkflowTemplateShortName:
		wf = common.NewWorkflowFromWorkflowTemplate(req.ResourceName, false)
	case workflow.ClusterWorkflowTemplateKind, workflow.ClusterWorkflowTemplateSingular, workflow.ClusterWorkflowTemplatePlural, workflow.ClusterWorkflowTemplateShortName:
		wf = common.NewWorkflowFromWorkflowTemplate(req.ResourceName, true)
	default:
		err := errors.Errorf(errors.CodeBadRequest, "Resource kind '%s' is not supported for submitting", req.ResourceKind)
		err = sutils.ToStatusError(err, codes.InvalidArgument)
		return nil, err
	}

	s.instanceIDService.Label(wf)
	creator.Label(ctx, wf)
	err := util.ApplySubmitOpts(wf, req.SubmitOptions)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	err = validate.ValidateWorkflow(wftmplGetter, cwftmplGetter, wf, validate.ValidateOpts{Submit: true})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	wf, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	return wf, nil
}
