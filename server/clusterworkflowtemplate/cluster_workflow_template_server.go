package clusterworkflowtemplate

import (
	"context"
	"fmt"
	"sort"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterwftmplpkg "github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/instanceid"
	"github.com/argoproj/argo/workflow/creator"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/validate"
)

type ClusterWorkflowTemplateServer struct {
	instanceIDService instanceid.Service
}

func NewClusterWorkflowTemplateServer(instanceID instanceid.Service) clusterwftmplpkg.ClusterWorkflowTemplateServiceServer {
	return &ClusterWorkflowTemplateServer{instanceID}
}

func (cwts *ClusterWorkflowTemplateServer) CreateClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateCreateRequest) (*v1alpha1.ClusterWorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	if req.Template == nil {
		return nil, fmt.Errorf("cluster workflow template was not found in the request body")
	}
	cwts.instanceIDService.Label(req.Template)
	creator.Label(ctx, req.Template)
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
	_, err := validate.ValidateClusterWorkflowTemplate(nil, cwftmplGetter, req.Template)
	if err != nil {
		return nil, err
	}
	return wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().Create(req.Template)
}

func (cwts *ClusterWorkflowTemplateServer) GetClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateGetRequest) (*v1alpha1.ClusterWorkflowTemplate, error) {
	wfTmpl, err := cwts.getTemplateAndValidate(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return wfTmpl, nil
}

func (cwts *ClusterWorkflowTemplateServer) getTemplateAndValidate(ctx context.Context, name string) (*v1alpha1.ClusterWorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	wfTmpl, err := wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().Get(name, v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = cwts.instanceIDService.Validate(wfTmpl)
	if err != nil {
		return nil, err
	}
	return wfTmpl, nil
}

func (cwts *ClusterWorkflowTemplateServer) ListClusterWorkflowTemplates(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateListRequest) (*v1alpha1.ClusterWorkflowTemplateList, error) {
	wfClient := auth.GetWfClient(ctx)
	options := &v1.ListOptions{}
	if req.ListOptions != nil {
		options = req.ListOptions
	}
	cwts.instanceIDService.With(options)
	cwfList, err := wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().List(*options)
	if err != nil {
		return nil, err
	}

	sort.Sort(cwfList.Items)

	return cwfList, nil
}

func (cwts *ClusterWorkflowTemplateServer) DeleteClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateDeleteRequest) (*clusterwftmplpkg.ClusterWorkflowTemplateDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)
	_, err := cwts.getTemplateAndValidate(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	err = wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().Delete(req.Name, &v1.DeleteOptions{})
	if err != nil {
		return nil, err
	}

	return &clusterwftmplpkg.ClusterWorkflowTemplateDeleteResponse{}, nil
}

func (cwts *ClusterWorkflowTemplateServer) LintClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateLintRequest) (*v1alpha1.ClusterWorkflowTemplate, error) {
	cwts.instanceIDService.Label(req.Template)
	creator.Label(ctx, req.Template)
	wfClient := auth.GetWfClient(ctx)
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	_, err := validate.ValidateClusterWorkflowTemplate(nil, cwftmplGetter, req.Template)
	if err != nil {
		return nil, err
	}

	return req.Template, nil
}

func (cwts *ClusterWorkflowTemplateServer) UpdateClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateUpdateRequest) (*v1alpha1.ClusterWorkflowTemplate, error) {
	if req.Template == nil {
		return nil, fmt.Errorf("ClusterWorkflowTemplate is not found in Request body")
	}
	err := cwts.instanceIDService.Validate(req.Template)
	if err != nil {
		return nil, err
	}
	wfClient := auth.GetWfClient(ctx)
	cwftmplGetter := templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())

	_, err = validate.ValidateClusterWorkflowTemplate(nil, cwftmplGetter, req.Template)
	if err != nil {
		return nil, err
	}

	res, err := wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().Update(req.Template)
	return res, err
}
