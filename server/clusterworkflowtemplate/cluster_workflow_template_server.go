package clusterworkflowtemplate

import (
	"context"
	"fmt"
	"sort"

	"google.golang.org/grpc/codes"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterwftmplpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	servertypes "github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"

	serverutils "github.com/argoproj/argo-workflows/v3/server/utils"
)

type ClusterWorkflowTemplateServer struct {
	instanceIDService instanceid.Service
	cwftmplStore      servertypes.ClusterWorkflowTemplateStore
}

func NewClusterWorkflowTemplateServer(instanceID instanceid.Service, cwftmplStore servertypes.ClusterWorkflowTemplateStore) clusterwftmplpkg.ClusterWorkflowTemplateServiceServer {
	return &ClusterWorkflowTemplateServer{instanceID, cwftmplStore}
}

func (cwts *ClusterWorkflowTemplateServer) cwftmplGetter(wfClient versioned.Interface) templateresolution.ClusterWorkflowTemplateGetter {
	if cwts.cwftmplStore != nil {
		return cwts.cwftmplStore.Getter()
	}
	return templateresolution.WrapClusterWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates())
}

func (cwts *ClusterWorkflowTemplateServer) CreateClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateCreateRequest) (*v1alpha1.ClusterWorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	if req.Template == nil {
		return nil, serverutils.ToStatusError(fmt.Errorf("cluster workflow template was not found in the request body"), codes.InvalidArgument)
	}
	cwts.instanceIDService.Label(req.Template)
	creator.Label(ctx, req.Template)
	cwftmplGetter := cwts.cwftmplGetter(wfClient)
	err := validate.ValidateClusterWorkflowTemplate(nil, cwftmplGetter, req.Template, validate.ValidateOpts{})
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.InvalidArgument)
	}
	res, err := wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().Create(ctx, req.Template, v1.CreateOptions{})
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.Internal)
	}
	return res, nil
}

func (cwts *ClusterWorkflowTemplateServer) GetClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateGetRequest) (*v1alpha1.ClusterWorkflowTemplate, error) {
	wfTmpl, err := cwts.getTemplateAndValidate(ctx, req.Name)
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.Internal)
	}
	return wfTmpl, nil
}

func (cwts *ClusterWorkflowTemplateServer) getTemplateAndValidate(ctx context.Context, name string) (*v1alpha1.ClusterWorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	wfTmpl, err := wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.Internal)
	}
	err = cwts.instanceIDService.Validate(wfTmpl)
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.InvalidArgument)
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
	cwfList, err := wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().List(ctx, *options)
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.Internal)
	}

	sort.Sort(cwfList.Items)

	return cwfList, nil
}

func (cwts *ClusterWorkflowTemplateServer) DeleteClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateDeleteRequest) (*clusterwftmplpkg.ClusterWorkflowTemplateDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)
	_, err := cwts.getTemplateAndValidate(ctx, req.Name)
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.InvalidArgument)
	}
	err = wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().Delete(ctx, req.Name, v1.DeleteOptions{})
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.Internal)
	}

	return &clusterwftmplpkg.ClusterWorkflowTemplateDeleteResponse{}, nil
}

func (cwts *ClusterWorkflowTemplateServer) LintClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateLintRequest) (*v1alpha1.ClusterWorkflowTemplate, error) {
	cwts.instanceIDService.Label(req.Template)
	creator.Label(ctx, req.Template)
	wfClient := auth.GetWfClient(ctx)
	cwftmplGetter := cwts.cwftmplGetter(wfClient)

	err := validate.ValidateClusterWorkflowTemplate(nil, cwftmplGetter, req.Template, validate.ValidateOpts{Lint: true})
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.InvalidArgument)
	}

	return req.Template, nil
}

func (cwts *ClusterWorkflowTemplateServer) UpdateClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateUpdateRequest) (*v1alpha1.ClusterWorkflowTemplate, error) {
	if req.Template == nil {
		return nil, serverutils.ToStatusError(fmt.Errorf("ClusterWorkflowTemplate is not found in Request body"), codes.InvalidArgument)
	}
	err := cwts.instanceIDService.Validate(req.Template)
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.InvalidArgument)
	}
	wfClient := auth.GetWfClient(ctx)
	cwftmplGetter := cwts.cwftmplGetter(wfClient)

	err = validate.ValidateClusterWorkflowTemplate(nil, cwftmplGetter, req.Template, validate.ValidateOpts{})
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.InvalidArgument)
	}

	res, err := wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().Update(ctx, req.Template, v1.UpdateOptions{})
	return res, serverutils.ToStatusError(err, codes.Internal)
}
