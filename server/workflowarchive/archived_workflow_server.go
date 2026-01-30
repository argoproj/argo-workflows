package workflowarchive

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v3/workflow/util"

	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
)

const disableValueListRetrievalKeyPattern = "DISABLE_VALUE_LIST_RETRIEVAL_KEY_PATTERN"

type archivedWorkflowServer struct {
	wfArchive             sqldb.WorkflowArchive
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	hydrator              hydrator.Interface
	wfDefaults            *wfv1.Workflow
}

// NewWorkflowArchiveServer returns a new archivedWorkflowServer
func NewWorkflowArchiveServer(wfArchive sqldb.WorkflowArchive, offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo, wfDefaults *wfv1.Workflow) workflowarchivepkg.ArchivedWorkflowServiceServer {
	return &archivedWorkflowServer{wfArchive, offloadNodeStatusRepo, hydrator.New(offloadNodeStatusRepo), wfDefaults}
}

func (w *archivedWorkflowServer) ListArchivedWorkflows(ctx context.Context, req *workflowarchivepkg.ListArchivedWorkflowsRequest) (*wfv1.WorkflowList, error) {
	listOptions := metav1.ListOptions{}
	if req.ListOptions != nil {
		listOptions = *req.ListOptions
	}

	options, err := sutils.BuildListOptions(listOptions, req.Namespace, req.NamePrefix, req.NameFilter, "", "")
	if err != nil {
		return nil, err
	}

	// verify if we have permission to list Workflows
	allowed, err := auth.CanI(ctx, "list", workflow.WorkflowPlural, options.Namespace, "")
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("Permission denied, you are not allowed to list workflows in namespace \"%s\". Maybe you want to specify a namespace with query parameter `.namespace=%s`?", options.Namespace, options.Namespace))
	}

	limit := options.Limit
	offset := options.Offset
	// When the zero value is passed, we should treat this as returning all results
	// to align ourselves with the behavior of the `List` endpoints in the Kubernetes API
	loadAll := limit == 0

	if !loadAll {
		// Attempt to load 1 more record than we actually need as an easy way to determine whether or not more
		// records exist than we're currently requesting
		options.Limit++
	}

	items, err := w.wfArchive.ListWorkflows(ctx, options)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	meta := metav1.ListMeta{}

	if options.ShowRemainingItemCount && !loadAll {
		total, err := w.wfArchive.CountWorkflows(ctx, options)
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		count := total - int64(offset) - int64(items.Len())
		if len(items) > limit {
			count++
		}
		if count < 0 {
			count = 0
		}
		meta.RemainingItemCount = &count
	}

	if !loadAll && len(items) > limit {
		items = items[0:limit]
		meta.Continue = fmt.Sprintf("%v", offset+limit)
	}

	sort.Sort(items)
	return &wfv1.WorkflowList{ListMeta: meta, Items: items}, nil
}

func (w *archivedWorkflowServer) GetArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.GetArchivedWorkflowRequest) (*wfv1.Workflow, error) {
	wf, err := w.wfArchive.GetWorkflow(ctx, req.Uid, req.Namespace, req.Name)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if wf == nil {
		// no need to call ToStatusError since it is already a status
		return nil, status.Error(codes.NotFound, "not found")
	}
	allowed, err := auth.CanI(ctx, "get", workflow.WorkflowPlural, wf.Namespace, wf.Name)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return wf, nil
}

func (w *archivedWorkflowServer) DeleteArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.DeleteArchivedWorkflowRequest) (*workflowarchivepkg.ArchivedWorkflowDeletedResponse, error) {
	wf, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: req.Uid})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	allowed, err := auth.CanI(ctx, "delete", workflow.WorkflowPlural, wf.Namespace, wf.Name)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if !allowed {
		// no need for ToStatusError since it is already the same time
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	err = w.wfArchive.DeleteWorkflow(ctx, req.Uid)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return &workflowarchivepkg.ArchivedWorkflowDeletedResponse{}, nil
}

func (w *archivedWorkflowServer) ListArchivedWorkflowLabelKeys(ctx context.Context, req *workflowarchivepkg.ListArchivedWorkflowLabelKeysRequest) (*wfv1.LabelKeys, error) {
	labelkeys, err := w.wfArchive.ListWorkflowsLabelKeys(ctx)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return labelkeys, nil
}

func matchLabelKeyPattern(key string) bool {
	pattern, _ := os.LookupEnv(disableValueListRetrievalKeyPattern)
	if pattern == "" {
		return false
	}
	match, _ := regexp.MatchString(pattern, key)
	return match
}

func (w *archivedWorkflowServer) ListArchivedWorkflowLabelValues(ctx context.Context, req *workflowarchivepkg.ListArchivedWorkflowLabelValuesRequest) (*wfv1.LabelValues, error) {
	options := req.ListOptions

	requirements, err := labels.ParseToRequirements(options.LabelSelector)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	if len(requirements) != 1 {
		return nil, sutils.ToStatusError(fmt.Errorf("only allow 1 labelRequirement, found %v", len(requirements)), codes.InvalidArgument)
	}

	requirement := requirements[0]
	if requirement.Operator() != selection.Exists {
		return nil, sutils.ToStatusError(fmt.Errorf("operation %v is not supported", requirement.Operator()), codes.InvalidArgument)
	}
	key := requirement.Key()
	if matchLabelKeyPattern(key) {
		logging.RequireLoggerFromContext(ctx).WithField("labelKey", key).Info(ctx, "Skipping retrieving the list of values for label key")
		return &wfv1.LabelValues{Items: []string{}}, nil
	}

	labels, err := w.wfArchive.ListWorkflowsLabelValues(ctx, key)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if labels == nil {
		// already a status so no need for ToStatusError
		return nil, status.Error(codes.NotFound, "not found")
	}
	return labels, nil
}

func (w *archivedWorkflowServer) ResubmitArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.ResubmitArchivedWorkflowRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wf, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: req.Uid})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	newWF, err := util.FormulateResubmitWorkflow(ctx, wf, req.Memoized, req.Parameters)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	creator.LabelCreator(ctx, newWF)

	created, err := util.SubmitWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wfClient, req.Namespace, newWF, w.wfDefaults, &wfv1.SubmitOpts{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return created, nil
}

func (w *archivedWorkflowServer) RetryArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.RetryArchivedWorkflowRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)

	wf, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: req.Uid})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	oriUID := wf.UID

	_, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if apierr.IsNotFound(err) {

		wf, podsToDelete, err := util.FormulateRetryWorkflow(ctx, wf, req.RestartSuccessful, req.NodeFieldSelector, req.Parameters)
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}

		logger := logging.RequireLoggerFromContext(ctx)
		for _, podName := range podsToDelete {
			logger.WithField("podDeleted", podName).Info(ctx, "Deleting pod")
			err := kubeClient.CoreV1().Pods(wf.Namespace).Delete(ctx, podName, metav1.DeleteOptions{})
			if err != nil && !apierr.IsNotFound(err) {
				return nil, sutils.ToStatusError(err, codes.Internal)
			}
		}

		logger.WithField("Dehydrate workflow uid=", wf.UID).Info(ctx, "RetryArchivedWorkflow")
		// If the Workflow needs to be dehydrated in order to capture and retain all of the previous state for the subsequent workflow, then do so
		err = w.hydrator.Dehydrate(ctx, wf)
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}

		wf.ResourceVersion = ""
		wf.UID = ""
		result, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		// if the Workflow was dehydrated before, we need to capture and maintain its previous state for the new Workflow
		if !w.hydrator.IsHydrated(wf) {
			offloadedNodes, err := w.offloadNodeStatusRepo.Get(ctx, string(oriUID), wf.GetOffloadNodeStatusVersion())
			if err != nil {
				return nil, sutils.ToStatusError(err, codes.Internal)
			}
			_, err = w.offloadNodeStatusRepo.Save(ctx, string(result.UID), wf.Namespace, offloadedNodes)
			if err != nil {
				return nil, sutils.ToStatusError(err, codes.Internal)
			}
		}

		return result, nil
	}

	if err == nil {
		// no need for ToStatusError since error is already status
		return nil, status.Error(codes.AlreadyExists, "Workflow already exists on cluster, use argo retry {name} instead")
	}

	return nil, sutils.ToStatusError(err, codes.Internal)
}
