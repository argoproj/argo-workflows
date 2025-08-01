package clusterworkflowtemplate

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/expr-lang/expr"
	"google.golang.org/grpc/codes"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusterwftmplpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/server/auth"
	servertypes "github.com/argoproj/argo-workflows/v3/server/types"
	"github.com/argoproj/argo-workflows/v3/util/expr/env"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/creator"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"

	serverutils "github.com/argoproj/argo-workflows/v3/server/utils"
)

type ClusterWorkflowTemplateServer struct {
	instanceIDService instanceid.Service
	cwftmplStore      servertypes.ClusterWorkflowTemplateStore
	wfDefaults        *v1alpha1.Workflow
}

func NewClusterWorkflowTemplateServer(instanceID instanceid.Service, cwftmplStore servertypes.ClusterWorkflowTemplateStore, wfDefaults *v1alpha1.Workflow) clusterwftmplpkg.ClusterWorkflowTemplateServiceServer {
	if cwftmplStore == nil {
		cwftmplStore = NewClusterWorkflowTemplateClientStore()
	}
	return &ClusterWorkflowTemplateServer{instanceID, cwftmplStore, wfDefaults}
}

func (cwts *ClusterWorkflowTemplateServer) CreateClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateCreateRequest) (*v1alpha1.ClusterWorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	if req.Template == nil {
		return nil, serverutils.ToStatusError(fmt.Errorf("cluster workflow template was not found in the request body"), codes.InvalidArgument)
	}
	cwts.instanceIDService.Label(req.Template)
	creator.LabelCreator(ctx, req.Template)
	cwftmplGetter := cwts.cwftmplStore.Getter(ctx)
	err := validate.ValidateClusterWorkflowTemplate(ctx, nil, cwftmplGetter, req.Template, cwts.wfDefaults, validate.ValidateOpts{})
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

	// 1. Evaluate global parameters first. They cannot reference other parameters.
	globalParamsMap := make(map[string]interface{})
	globalEvalEnv := env.GetFuncMap(nil) // Env with just sprig functions

	for i, param := range wfTmpl.Spec.Arguments.Parameters {
		if param.Default != nil {
			paramStr := strings.TrimSpace(param.Default.String())
			if strings.HasPrefix(paramStr, "{{=") && strings.HasSuffix(paramStr, "}}") {
				expression := strings.TrimSpace(paramStr[3 : len(paramStr)-2])
				val, err := expr.Eval(expression, globalEvalEnv)
				if err == nil {
					newValue := v1alpha1.ParseAnyString(val)
					wfTmpl.Spec.Arguments.Parameters[i].Value = &newValue
					globalParamsMap[param.Name] = val
				}
			} else {
				// This is a static default value, not an expression.
				globalParamsMap[param.Name] = param.Default.String()
			}
		}
	}

	// 2. Evaluate template-level parameters.
	// The environment for these parameters includes the evaluated global parameters.
	templateEvalEnv := env.GetFuncMap(nil)
	templateEvalEnv["workflow"] = map[string]interface{}{
		"parameters": globalParamsMap,
	}

	for i, tmpl := range wfTmpl.Spec.Templates {
		for j, param := range tmpl.Inputs.Parameters {
			if param.Default != nil {
				paramStr := strings.TrimSpace(param.Default.String())
				if strings.HasPrefix(paramStr, "{{=") && strings.HasSuffix(paramStr, "}}") {
					expression := strings.TrimSpace(paramStr[3 : len(paramStr)-2])
					val, err := expr.Eval(expression, templateEvalEnv)
					if err == nil {
						newValue := v1alpha1.ParseAnyString(val)
						wfTmpl.Spec.Templates[i].Inputs.Parameters[j].Value = &newValue
					}
				}
			}
		}
	}

	return wfTmpl, nil
}

func (cwts *ClusterWorkflowTemplateServer) getTemplateAndValidate(ctx context.Context, name string) (*v1alpha1.ClusterWorkflowTemplate, error) {
	wfTmpl, err := cwts.cwftmplStore.Getter(ctx).Get(ctx, name)
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
	creator.LabelCreator(ctx, req.Template)
	cwftmplGetter := cwts.cwftmplStore.Getter(ctx)

	err := validate.ValidateClusterWorkflowTemplate(ctx, nil, cwftmplGetter, req.Template, cwts.wfDefaults, validate.ValidateOpts{Lint: true})
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.InvalidArgument)
	}

	return req.Template, nil
}

func (cwts *ClusterWorkflowTemplateServer) UpdateClusterWorkflowTemplate(ctx context.Context, req *clusterwftmplpkg.ClusterWorkflowTemplateUpdateRequest) (*v1alpha1.ClusterWorkflowTemplate, error) {
	if req.Template == nil {
		return nil, serverutils.ToStatusError(fmt.Errorf("ClusterWorkflowTemplate is not found in Request body"), codes.InvalidArgument)
	}
	creator.LabelActor(ctx, req.Template, creator.ActionUpdate)
	err := cwts.instanceIDService.Validate(req.Template)
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.InvalidArgument)
	}
	wfClient := auth.GetWfClient(ctx)
	cwftmplGetter := cwts.cwftmplStore.Getter(ctx)

	err = validate.ValidateClusterWorkflowTemplate(ctx, nil, cwftmplGetter, req.Template, cwts.wfDefaults, validate.ValidateOpts{})
	if err != nil {
		return nil, serverutils.ToStatusError(err, codes.InvalidArgument)
	}

	res, err := wfClient.ArgoprojV1alpha1().ClusterWorkflowTemplates().Update(ctx, req.Template, v1.UpdateOptions{})
	return res, serverutils.ToStatusError(err, codes.Internal)
}
