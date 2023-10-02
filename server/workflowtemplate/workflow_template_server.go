package workflowtemplate

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"google.golang.org/grpc/codes"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
)

type WorkflowTemplateServer struct {
	instanceIDService instanceid.Service
}

func NewWorkflowTemplateServer(instanceIDService instanceid.Service) workflowtemplatepkg.WorkflowTemplateServiceServer {
	return &WorkflowTemplateServer{instanceIDService}
}

func (wts *WorkflowTemplateServer) CreateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateCreateRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	if req.Template == nil {
		return nil, sutils.ToStatusError(fmt.Errorf("workflow template was not found in the request body"), codes.InvalidArgument)
	}
	wts.instanceIDService.Label(req.Template)
	creator.Label(ctx, req.Template)
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	err := validate.ValidateWorkflowTemplate(wftmplGetter, cwftmplGetter, req.Template, validate.ValidateOpts{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	wfTmpl, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Create(ctx, req.Template, v1.CreateOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	return wfTmpl, nil
}

func (wts *WorkflowTemplateServer) GetWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateGetRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfTmpl, err := wts.getTemplateAndValidate(ctx, req.Namespace, req.Name)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return wfTmpl, nil
}

func (wts *WorkflowTemplateServer) getTemplateAndValidate(ctx context.Context, namespace string, name string) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	wfTmpl, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace).Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	err = wts.instanceIDService.Validate(wfTmpl)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	return wfTmpl, nil
}

func cursorPaginationByResourceVersion(items []v1alpha1.WorkflowTemplate, resourceVersion string, limit int64, wfList *v1alpha1.WorkflowTemplateList) {
	// Sort the workflow list in descending order by resourceVersion.
	sort.Slice(items, func(i, j int) bool {
		itemIRV, _ := strconv.Atoi(items[i].ResourceVersion)
		itemJRV, _ := strconv.Atoi(items[j].ResourceVersion)
		return itemIRV > itemJRV
	})

	// resourceVersion: unique value to identify the version of the object by Kubernetes. It is used for pagination in workflows.
	// receivedRV: resourceVersion value used for previous pagination
	// Due to the descending sorting above, the items are filtered to have a resourceVersion smaller than receivedRV.
	// The data with values smaller than the receivedRV on the current page will be used for the next page.
	if resourceVersion != "" {
		var newItems []v1alpha1.WorkflowTemplate
		for _, item := range items {
			targetRV, _ := strconv.Atoi(item.ResourceVersion)
			receivedRV, _ := strconv.Atoi(resourceVersion)
			if targetRV < receivedRV {
				newItems = append(newItems, item)
			}
			items = newItems
		}
	}

	// Indexing list by limit count
	if limit != 0 {
		endIndex := int(limit)
		if endIndex > len(items) || limit == 0 {
			endIndex = len(items)
		}
		wfList.Items = items[0:endIndex]
	} else {
		wfList.Items = items
	}

	// Calculate new offset for next page
	// For the next pagination, the resourceVersion of the last item is set in the Continue field.
	if limit != 0 && len(wfList.Items) == int(limit) {
		lastIndex := len(wfList.Items) - 1
		wfList.ListMeta.Continue = wfList.Items[lastIndex].ResourceVersion
	}
}

func (wts *WorkflowTemplateServer) ListWorkflowTemplates(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateListRequest) (*v1alpha1.WorkflowTemplateList, error) {
	wfClient := auth.GetWfClient(ctx)
	k8sOptions := &v1.ListOptions{}

	if req.ListOptions != nil {
		k8sOptions = req.ListOptions
	}

	// Save the original Continue and Limit for custom filtering.
	resourceVersion := k8sOptions.Continue
	limit := k8sOptions.Limit

	if req.NamePattern != "" {
		// Search whole with Limit 0.
		// Reset the Continue "" to prevent Kubernetes native pagination.
		// Kubernetes api will search for all results without limit and pagination.
		k8sOptions.Continue = ""
		k8sOptions.Limit = 0
	}

	wts.instanceIDService.With(k8sOptions)
	wfList, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).List(ctx, *k8sOptions)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}

	// Enables name-based searching.
	// Do name pattern filtering if NamePattern request exist.
	var items []v1alpha1.WorkflowTemplate
	if req.NamePattern != "" {
		for _, item := range wfList.Items {
			if strings.Contains(item.ObjectMeta.Name, req.NamePattern) {
				items = append(items, item)
			}
		}
		cursorPaginationByResourceVersion(items, resourceVersion, limit, wfList)
	}

	sort.Sort(wfList.Items)
	return wfList, nil
}

func (wts *WorkflowTemplateServer) DeleteWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateDeleteRequest) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)
	_, err := wts.getTemplateAndValidate(ctx, req.Namespace, req.Name)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	err = wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Delete(ctx, req.Name, v1.DeleteOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return &workflowtemplatepkg.WorkflowTemplateDeleteResponse{}, nil
}

func (wts *WorkflowTemplateServer) LintWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateLintRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	wts.instanceIDService.Label(req.Template)
	creator.Label(ctx, req.Template)
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	err := validate.ValidateWorkflowTemplate(wftmplGetter, cwftmplGetter, req.Template, validate.ValidateOpts{Lint: true})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	return req.Template, nil
}

func (wts *WorkflowTemplateServer) UpdateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateUpdateRequest) (*v1alpha1.WorkflowTemplate, error) {
	if req.Template == nil {
		return nil, sutils.ToStatusError(fmt.Errorf("WorkflowTemplate is not found in Request body"), codes.InvalidArgument)
	}
	err := wts.instanceIDService.Validate(req.Template)
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	wfClient := auth.GetWfClient(ctx)
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	err = validate.ValidateWorkflowTemplate(wftmplGetter, cwftmplGetter, req.Template, validate.ValidateOpts{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.InvalidArgument)
	}
	res, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Update(ctx, req.Template, v1.UpdateOptions{})
	if err != nil {
		return nil, sutils.ToStatusError(err, codes.Internal)
	}
	return res, nil
}
