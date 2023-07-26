package workflowarchive

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
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
	"github.com/argoproj/argo-workflows/v3/workflow/util"

	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
)

const disableValueListRetrievalKeyPattern = "DISABLE_VALUE_LIST_RETRIEVAL_KEY_PATTERN"

type archivedWorkflowServer struct {
	wfArchive sqldb.WorkflowArchive
}

// NewWorkflowArchiveServer returns a new archivedWorkflowServer
func NewWorkflowArchiveServer(wfArchive sqldb.WorkflowArchive) workflowarchivepkg.ArchivedWorkflowServiceServer {
	return &archivedWorkflowServer{wfArchive: wfArchive}
}

func (w *archivedWorkflowServer) ListArchivedWorkflows(ctx context.Context, req *workflowarchivepkg.ListArchivedWorkflowsRequest) (*wfv1.WorkflowList, error) {
	options := req.ListOptions
	namePrefix := req.NamePrefix
	if options == nil {
		options = &metav1.ListOptions{}
	}
	if options.Continue == "" {
		options.Continue = "0"
	}
	limit := int(options.Limit)

	offset, err := strconv.Atoi(options.Continue)
	if err != nil {
		// no need to use sutils here
		return nil, status.Error(codes.InvalidArgument, "listOptions.continue must be int")
	}
	if offset < 0 {
		// no need to use sutils here
		return nil, status.Error(codes.InvalidArgument, "listOptions.continue must >= 0")
	}

	// namespace is now specified as its own query parameter
	// note that for backward compatibility, the field selector 'metadata.namespace' is also supported for now
	namespace := req.Namespace // optional
	name := ""
	minStartedAt := time.Time{}
	maxStartedAt := time.Time{}
	showRemainingItemCount := false
	for _, selector := range strings.Split(options.FieldSelector, ",") {
		if len(selector) == 0 {
			continue
		}
		if strings.HasPrefix(selector, "metadata.namespace=") {
			// for backward compatibility, the field selector 'metadata.namespace' is supported for now despite the addition
			// of the new 'namespace' query parameter, which is what the UI uses
			fieldSelectedNamespace := strings.TrimPrefix(selector, "metadata.namespace=")
			switch namespace {
			case "":
				namespace = fieldSelectedNamespace
			case fieldSelectedNamespace:
				break
			default:
				return nil, status.Errorf(codes.InvalidArgument,
					"'namespace' query param (%q) and fieldselector 'metadata.namespace' (%q) are both specified and contradict each other", namespace, fieldSelectedNamespace)
			}
		} else if strings.HasPrefix(selector, "metadata.name=") {
			name = strings.TrimPrefix(selector, "metadata.name=")
		} else if strings.HasPrefix(selector, "spec.startedAt>") {
			minStartedAt, err = time.Parse(time.RFC3339, strings.TrimPrefix(selector, "spec.startedAt>"))
			if err != nil {
				// startedAt is populated by us, it should therefore be valid.
				return nil, sutils.ToStatusError(err, codes.Internal)
			}
		} else if strings.HasPrefix(selector, "spec.startedAt<") {
			maxStartedAt, err = time.Parse(time.RFC3339, strings.TrimPrefix(selector, "spec.startedAt<"))
			if err != nil {
				// no need to use sutils here
				return nil, sutils.ToStatusError(err, codes.Internal)
			}
		} else if strings.HasPrefix(selector, "ext.showRemainingItemCount") {
			showRemainingItemCount, err = strconv.ParseBool(strings.TrimPrefix(selector, "ext.showRemainingItemCount="))
			if err != nil {
				// populated by us, it should therefore be valid.
				return nil, sutils.ToStatusError(err, codes.Internal)
			}
		} else {
			return nil, sutils.ToStatusError(fmt.Errorf("unsupported requirement %s", selector), codes.InvalidArgument)
		}
	}
	requirements, err := labels.ParseToRequirements(options.LabelSelector)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}

	// verify if we have permission to list Workflows
	allowed, err := auth.CanI(ctx, "list", workflow.WorkflowPlural, namespace, "")
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("Permission denied, you are not allowed to list workflows in namespace \"%s\". Maybe you want to specify a namespace with query parameter `.namespace=%s`?", namespace, namespace))
	}

	// When the zero value is passed, we should treat this as returning all results
	// to align ourselves with the behavior of the `List` endpoints in the Kubernetes API
	loadAll := limit == 0
	limitWithMore := 0

	if !loadAll {
		// Attempt to load 1 more record than we actually need as an easy way to determine whether or not more
		// records exist than we're currently requesting
		limitWithMore = limit + 1
	}

	items, err := w.wfArchive.ListWorkflows(namespace, name, namePrefix, minStartedAt, maxStartedAt, requirements, limitWithMore, offset)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	meta := metav1.ListMeta{}

	if showRemainingItemCount && !loadAll {
		total, err := w.wfArchive.CountWorkflows(namespace, name, namePrefix, minStartedAt, maxStartedAt, requirements)
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		count := total - int64(offset) - int64(items.Len())
		if len(items) > limit {
			count = count + 1
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
	wf, err := w.wfArchive.GetWorkflow(req.Uid, req.Namespace, req.Name)
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
	err = w.wfArchive.DeleteWorkflow(req.Uid)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return &workflowarchivepkg.ArchivedWorkflowDeletedResponse{}, nil
}

func (w *archivedWorkflowServer) ListArchivedWorkflowLabelKeys(ctx context.Context, req *workflowarchivepkg.ListArchivedWorkflowLabelKeysRequest) (*wfv1.LabelKeys, error) {
	labelkeys, err := w.wfArchive.ListWorkflowsLabelKeys()
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

	key := ""
	requirement := requirements[0]
	if requirement.Operator() == selection.Exists {
		key = requirement.Key()
	} else {
		return nil, sutils.ToStatusError(fmt.Errorf("operation %v is not supported", requirement.Operator()), codes.InvalidArgument)
	}
	if matchLabelKeyPattern(key) {
		log.WithFields(log.Fields{"labelKey": key}).Info("Skipping retrieving the list of values for label key")
		return &wfv1.LabelValues{Items: []string{}}, nil
	}

	labels, err := w.wfArchive.ListWorkflowsLabelValues(key)
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

	created, err := util.SubmitWorkflow(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wfClient, req.Namespace, newWF, &wfv1.SubmitOpts{})
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

	_, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if apierr.IsNotFound(err) {

		wf, podsToDelete, err := util.FormulateRetryWorkflow(ctx, wf, req.RestartSuccessful, req.NodeFieldSelector, req.Parameters)
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}

		for _, podName := range podsToDelete {
			log.WithFields(log.Fields{"podDeleted": podName}).Info("Deleting pod")
			err := kubeClient.CoreV1().Pods(wf.Namespace).Delete(ctx, podName, metav1.DeleteOptions{})
			if err != nil && !apierr.IsNotFound(err) {
				return nil, sutils.ToStatusError(err, codes.Internal)
			}
		}

		wf.ObjectMeta.ResourceVersion = ""
		wf.ObjectMeta.UID = ""
		result, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}

		return result, nil
	}

	if err == nil {
		// no need for ToStatusError since error is already status
		return nil, status.Error(codes.AlreadyExists, "Workflow already exists on cluster, use argo retry {name} instead")
	}

	return nil, sutils.ToStatusError(err, codes.Internal)
}
