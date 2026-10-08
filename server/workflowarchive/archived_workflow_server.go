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

	"github.com/argoproj/argo-workflows/v4/persist/sqldb"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/server/auth"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/creator"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v4/workflow/util"

	sutils "github.com/argoproj/argo-workflows/v4/server/utils"
)

const disableValueListRetrievalKeyPattern = "DISABLE_VALUE_LIST_RETRIEVAL_KEY_PATTERN"

type archivedWorkflowServer struct {
	wfArchive              sqldb.WorkflowArchive
	offloadNodeStatusRepo  sqldb.OffloadNodeStatusRepo
	hydrator               hydrator.Interface
	wfDefaults             *wfv1.Workflow
	templateRepo           sqldb.TemplateRepo
	templateOffloadMinSize int
	templateOffload        bool
}

// NewWorkflowArchiveServer returns a new archivedWorkflowServer
func NewWorkflowArchiveServer(wfArchive sqldb.WorkflowArchive, offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo, wfDefaults *wfv1.Workflow, templateRepo sqldb.TemplateRepo, templateOffloadMinSize int, templateOffload bool) workflowarchivepkg.ArchivedWorkflowServiceServer {
	return &archivedWorkflowServer{wfArchive, offloadNodeStatusRepo, hydrator.New(offloadNodeStatusRepo), wfDefaults, templateRepo, templateOffloadMinSize, templateOffload}
}

// templateRepoForOffload returns the repository only when this server may create new
// offloads. The repository itself stays available for hydration and cleanup when
// templateOffLoad is disabled.
func (w *archivedWorkflowServer) templateRepoForOffload() sqldb.TemplateRepo {
	if !w.templateOffload {
		return nil
	}
	return w.templateRepo
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
	// A negated namespace selector returns every other namespace, so it needs cluster-wide permission
	targetNamespace := options.Namespace
	if options.NamespaceFilter == "NotEquals" {
		targetNamespace = ""
	}
	allowed, err := auth.CanI(ctx, "list", workflow.WorkflowPlural, targetNamespace, "")
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("Permission denied, you are not allowed to list workflows in namespace \"%s\". Maybe you want to specify a namespace with query parameter `.namespace=%s`?", targetNamespace, targetNamespace))
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
	// Authorize before revealing whether the workflow exists. Answering "not found" without an
	// authorization check would let a caller without permission enumerate archived workflow
	// names from the difference between "not found" and "permission denied".
	namespace, name := req.Namespace, req.Name
	if wf != nil {
		namespace, name = wf.Namespace, wf.Name
	}
	allowed, err := auth.CanI(ctx, "get", workflow.WorkflowPlural, namespace, name)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	if !allowed {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}
	if wf == nil {
		// no need to call ToStatusError since it is already a status
		return nil, status.Error(codes.NotFound, "not found")
	}
	// Hydrate node status if offloaded
	if err := w.hydrator.Hydrate(ctx, wf); err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	// Hydrate templates if they were offloaded
	if err := w.hydrateTemplates(ctx, wf); err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return wf, nil
}

// hydrateTemplates loads templates from database if they were offloaded
func (w *archivedWorkflowServer) hydrateTemplates(ctx context.Context, wf *wfv1.Workflow) error {
	// Skip if template repo not configured
	if w.templateRepo == nil || !w.templateRepo.IsEnabled() {
		return nil
	}

	// Skip if templates were not offloaded for this workflow. The StoredTemplateSpecs
	// marker can be missing even when templates WERE offloaded (a lost resourceVersion
	// race) — WFT/ClusterWFT-ref workflows have no offload rows by design, so never probe.
	markerMissing := wf.Status.StoredTemplateSpecs == nil || wf.Status.StoredTemplateSpecs.UID == ""
	if markerMissing && wf.Spec.WorkflowTemplateRef != nil {
		return nil
	}

	// Skip if already hydrated (templates in spec)
	if len(wf.Spec.Templates) > 0 {
		return nil
	}

	// Only probe the DB when the archived record looks offloaded (empty spec templates).
	// Archived records carry templates at archive time now, so this is a fallback.
	if len(wf.Spec.Templates) == 0 {
		templates, err := w.templateRepo.GetTemplates(ctx, string(wf.UID))
		if err != nil {
			return fmt.Errorf("failed to hydrate templates: %w", err)
		}

		if len(templates) == 0 {
			logging.RequireLoggerFromContext(ctx).WithField("uid", wf.UID).Debug(ctx, "No offloaded templates found in database for archived workflow (may have been cleaned up)")
			return nil
		}

		// Populate templates in spec for API response
		wf.Spec.Templates = templates
		// Also populate StoredTemplates for template lookup
		if wf.Status.StoredTemplates == nil {
			wf.Status.StoredTemplates = make(map[string]wfv1.Template)
		}
		for _, tmpl := range templates {
			wf.Status.StoredTemplates[tmpl.Name] = tmpl
		}

		// Restore the marker on the hydrated record.
		if markerMissing {
			wf.Status.StoredTemplateSpecs = &wfv1.TemplateSpecReference{
				UID:      string(wf.UID),
				Version:  util.ComputeTemplateVersion(templates),
				Hydrated: true,
			}
		}
	}

	return nil
}

func (w *archivedWorkflowServer) DeleteArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.DeleteArchivedWorkflowRequest) (*workflowarchivepkg.ArchivedWorkflowDeletedResponse, error) {
	var wf *wfv1.Workflow
	var err error

	if req.Name != "" {
		wf, err = w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Name: req.Name, Namespace: req.Namespace})
	} else {
		wf, err = w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: req.Uid, Namespace: req.Namespace})
	}

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
	err = w.wfArchive.DeleteWorkflow(ctx, string(wf.UID))
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	// Clean up offloaded template rows so they don't become orphaned.
	// The workflow UID is still available from the archive record.
	if w.templateRepo != nil && w.templateRepo.IsEnabled() {
		if err := w.templateRepo.DeleteTemplates(ctx, string(wf.UID)); err != nil {
			// Log but don't fail the request — the archive record is already deleted
			// and the periodic template GC will eventually clean this up.
			logging.RequireLoggerFromContext(ctx).WithError(err).WithField("uid", wf.UID).Warn(ctx, "Failed to delete offloaded templates for archived workflow")
		}
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

	var wf *wfv1.Workflow
	var err error

	if req.Name != "" {
		wf, err = w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Name: req.Name, Namespace: req.Namespace})
	} else {
		wf, err = w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: req.Uid, Namespace: req.Namespace})
	}

	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	newWF, err := util.FormulateResubmitWorkflow(ctx, wf, req.Memoized, req.Parameters)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	creator.LabelCreator(ctx, newWF)

	created, err := util.SubmitWorkflowWithOffload(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wfClient, w.templateRepoForOffload(), req.Namespace, newWF, w.wfDefaults, &wfv1.SubmitOpts{}, w.templateOffloadMinSize)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return created, nil
}

func (w *archivedWorkflowServer) RetryArchivedWorkflow(ctx context.Context, req *workflowarchivepkg.RetryArchivedWorkflowRequest) (*wfv1.Workflow, error) {
	wfClient := auth.GetWfClient(ctx)
	kubeClient := auth.GetKubeClient(ctx)

	var wf *wfv1.Workflow
	var err error

	if req.Name != "" {
		wf, err = w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Name: req.Name, Namespace: req.Namespace})
	} else {
		wf, err = w.GetArchivedWorkflow(ctx, &workflowarchivepkg.GetArchivedWorkflowRequest{Uid: req.Uid, Namespace: req.Namespace})
	}

	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	oriUID := wf.UID

	_, err = wfClient.ArgoprojV1alpha1().Workflows(req.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	if apierr.IsNotFound(err) {
		var podsToDelete []string
		wf, podsToDelete, err = util.FormulateRetryWorkflow(ctx, wf, req.RestartSuccessful, req.NodeFieldSelector, req.Parameters)
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}

		logger := logging.RequireLoggerFromContext(ctx)
		for _, podName := range podsToDelete {
			logger.WithField("podDeleted", podName).Info(ctx, "Deleting pod")
			deleteErr := kubeClient.CoreV1().Pods(wf.Namespace).Delete(ctx, podName, metav1.DeleteOptions{})
			if deleteErr != nil && !apierr.IsNotFound(deleteErr) {
				return nil, sutils.ToStatusError(deleteErr, codes.Internal)
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
		wf.Status.StoredTemplateSpecs = nil
		wf.Status.StoredTemplates = nil
		result, createErr := util.CreateWorkflowWithOffload(ctx, wfClient.ArgoprojV1alpha1().Workflows(req.Namespace), wfClient, w.templateRepoForOffload(), wf, w.templateOffloadMinSize)
		if createErr != nil {
			return nil, sutils.ToStatusError(createErr, codes.Internal)
		}
		// if the Workflow was dehydrated before, we need to capture and maintain its previous state for the new Workflow
		if !w.hydrator.IsHydrated(wf) {
			offloadedNodes, getErr := w.offloadNodeStatusRepo.Get(ctx, string(oriUID), wf.GetOffloadNodeStatusVersion())
			if getErr != nil {
				return nil, sutils.ToStatusError(getErr, codes.Internal)
			}
			_, saveErr := w.offloadNodeStatusRepo.Save(ctx, string(result.UID), wf.Namespace, offloadedNodes)
			if saveErr != nil {
				return nil, sutils.ToStatusError(saveErr, codes.Internal)
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
