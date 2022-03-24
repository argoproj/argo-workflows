package workflowarchive

import (
	"context"
	"fmt"
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
)

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
		return nil, status.Error(codes.InvalidArgument, "listOptions.continue must be int")
	}
	if offset < 0 {
		return nil, status.Error(codes.InvalidArgument, "listOptions.continue must >= 0")
	}

	namespace := ""
	name := ""
	minStartedAt := time.Time{}
	maxStartedAt := time.Time{}
	for _, selector := range strings.Split(options.FieldSelector, ",") {
		if len(selector) == 0 {
			continue
		}
		if strings.HasPrefix(selector, "metadata.namespace=") {
			namespace = strings.TrimPrefix(selector, "metadata.namespace=")
		} else if strings.HasPrefix(selector, "metadata.name=") {
			name = strings.TrimPrefix(selector, "metadata.name=")
		} else if strings.HasPrefix(selector, "spec.startedAt>") {
			minStartedAt, err = time.Parse(time.RFC3339, strings.TrimPrefix(selector, "spec.startedAt>"))
			if err != nil {
				return nil, err
			}
		} else if strings.HasPrefix(selector, "spec.startedAt<") {
			maxStartedAt, err = time.Parse(time.RFC3339, strings.TrimPrefix(selector, "spec.startedAt<"))
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unsupported requirement %s", selector)
		}
	}
	requirements, err := labels.ParseToRequirements(options.LabelSelector)
	if err != nil {
		return nil, err
	}

	allowed, err := auth.CanI(ctx, "list", workflow.WorkflowPlural, namespace, "")
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("Permission denied, you are not allowed to list workflows in namespace \"%s\". Maybe you want to specify a namespace with `listOptions.fieldSelector=metadata.namespace=your-ns`?", namespace))
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
		return nil, err
	}

	meta := metav1.ListMeta{}

	if !loadAll && len(items) > limit {
		items = items[0:limit]
		meta.Continue = fmt.Sprintf("%v", offset+limit)
	}

	sort.Sort(items)
	return &wfv1.WorkflowList{ListMeta: meta, Items: items}, nil
}

func (w *archivedWorkflowServer) GetArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.GetArchivedWorkflowRequest) (*wfv1.Workflow, error) {
	wf, err := w.wfArchive.GetWorkflow(req.Uid)
	if err != nil {
		return nil, err
	}
	if wf == nil {
		return nil, status.Error(codes.NotFound, "not found")
	}
	allowed, err := auth.CanI(ctx, "get", workflow.WorkflowPlural, wf.Namespace, wf.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	return wf, err
}

func (w *archivedWorkflowServer) DeleteArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.DeleteArchivedWorkflowRequest) (*workflowarchivepkg.ArchivedWorkflowDeletedResponse, error) {
	wf, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: req.Uid})
	if err != nil {
		return nil, err
	}
	allowed, err := auth.CanI(ctx, "delete", workflow.WorkflowPlural, wf.Namespace, wf.Name)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	err = w.wfArchive.DeleteWorkflow(req.Uid)
	if err != nil {
		return nil, err
	}
	return &workflowarchivepkg.ArchivedWorkflowDeletedResponse{}, nil
}

func (w *archivedWorkflowServer) ListArchivedWorkflowLabelKeys(ctx context.Context, req *workflowarchivepkg.ListArchivedWorkflowLabelKeysRequest) (*wfv1.LabelKeys, error) {
	labelkeys, err := w.wfArchive.ListWorkflowsLabelKeys()
	if err != nil {
		return nil, err
	}
	return labelkeys, nil
}

func (w *archivedWorkflowServer) ListArchivedWorkflowLabelValues(ctx context.Context, req *workflowarchivepkg.ListArchivedWorkflowLabelValuesRequest) (*wfv1.LabelValues, error) {
	options := req.ListOptions

	requirements, err := labels.ParseToRequirements(options.LabelSelector)
	if err != nil {
		return nil, err
	}
	if len(requirements) != 1 {
		return nil, fmt.Errorf("only allow 1 labelRequirement, found %v", len(requirements))
	}

	key := ""
	requirement := requirements[0]
	if requirement.Operator() == selection.Exists {
		key = requirement.Key()
	} else {
		return nil, fmt.Errorf("operation %v is not supported", requirement.Operator())
	}

	labels, err := w.wfArchive.ListWorkflowsLabelValues(key)
	if err != nil {
		return nil, err
	}
	if labels == nil {
		return nil, status.Error(codes.NotFound, "not found")
	}
	return labels, err
}

func (w *archivedWorkflowServer) ResubmitArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.ResubmitArchivedWorkflowRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)

	wf, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: req.Uid})
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

func (w *archivedWorkflowServer) RetryArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.RetryArchivedWorkflowRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)

	wf, err := w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: req.Uid})
	if err != nil {
		return nil, err
	}

	_, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if apierr.IsNotFound(err) {

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

		wf.ObjectMeta.ResourceVersion = ""
		wf.ObjectMeta.UID = ""
		result, err := wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Create(ctx, wf, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	if err == nil {
		return nil, status.Error(codes.AlreadyExists, "Workflow already exists on cluster, use argo retry {name} instead")
	}

	return nil, err
}
