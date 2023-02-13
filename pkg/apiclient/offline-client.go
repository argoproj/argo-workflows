package apiclient

import (
	"context"
	"fmt"
	"os"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	infopkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/info"
	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	workflowarchivepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"

	"sigs.k8s.io/yaml"
)

type offlineClient struct {
	clusterWorkflowTemplateGetter       templateresolution.ClusterWorkflowTemplateGetter
	namespacedWorkflowTemplateGetterMap map[string]templateresolution.WorkflowTemplateNamespacedGetter
}

var OfflineErr = fmt.Errorf("not supported when you are in offline mode")

var _ Client = &offlineClient{}

func newOfflineClient(files []string) (context.Context, Client, error) {
	clusterWorkflowTemplateGetter := &offlineClusterWorkflowTemplateGetter{
		clusterWorkflowTemplates: map[string]*wfv1.ClusterWorkflowTemplate{},
	}
	workflowTemplateGetters := map[string]templateresolution.WorkflowTemplateNamespacedGetter{}

	for _, file := range files {
		bytes, err := os.ReadFile(file)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file %s: %w", file, err)
		}
		var generic map[string]interface{}
		if err := yaml.Unmarshal(bytes, &generic); err != nil {
			return nil, nil, fmt.Errorf("failed to parse YAML from file %s: %w", file, err)
		}
		switch generic["kind"] {
		case "ClusterWorkflowTemplate":
			cwftmpl := new(v1alpha1.ClusterWorkflowTemplate)
			if err := yaml.Unmarshal(bytes, &cwftmpl); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal file %s as a ClusterWorkflowTemplate: %w", file, err)
			}

			if _, ok := clusterWorkflowTemplateGetter.clusterWorkflowTemplates[cwftmpl.Name]; ok {
				return nil, nil, fmt.Errorf("duplicate ClusterWorkflowTemplate found: %q", cwftmpl.Name)
			}
			clusterWorkflowTemplateGetter.clusterWorkflowTemplates[cwftmpl.Name] = cwftmpl

		case "WorkflowTemplate":
			wftmpl := new(v1alpha1.WorkflowTemplate)
			if err := yaml.Unmarshal(bytes, &wftmpl); err != nil {
				return nil, nil, fmt.Errorf("failed to unmarshal file %s as a WorkflowTemplate: %w", file, err)
			}
			getter, ok := workflowTemplateGetters[wftmpl.Namespace]
			if !ok {
				getter = &offlineWorkflowTemplateNamespacedGetter{
					namespace:         wftmpl.Namespace,
					workflowTemplates: map[string]*wfv1.WorkflowTemplate{},
				}
				workflowTemplateGetters[wftmpl.Namespace] = getter
			}

			if _, ok := getter.(*offlineWorkflowTemplateNamespacedGetter).workflowTemplates[wftmpl.Name]; ok {
				return nil, nil, fmt.Errorf("duplicate WorkflowTemplate found: %q", wftmpl.Name)
			}
			getter.(*offlineWorkflowTemplateNamespacedGetter).workflowTemplates[wftmpl.Name] = wftmpl
		}

	}

	return context.Background(), &offlineClient{
		clusterWorkflowTemplateGetter:       clusterWorkflowTemplateGetter,
		namespacedWorkflowTemplateGetterMap: workflowTemplateGetters,
	}, nil
}

func (c *offlineClient) NewWorkflowServiceClient() workflowpkg.WorkflowServiceClient {
	return &errorTranslatingWorkflowServiceClient{OfflineWorkflowServiceClient{
		clusterWorkflowTemplateGetter:       c.clusterWorkflowTemplateGetter,
		namespacedWorkflowTemplateGetterMap: c.namespacedWorkflowTemplateGetterMap,
	}}
}

func (c *offlineClient) NewCronWorkflowServiceClient() (cronworkflow.CronWorkflowServiceClient, error) {
	return &errorTranslatingCronWorkflowServiceClient{OfflineCronWorkflowServiceClient{
		clusterWorkflowTemplateGetter:       c.clusterWorkflowTemplateGetter,
		namespacedWorkflowTemplateGetterMap: c.namespacedWorkflowTemplateGetterMap,
	}}, nil
}

func (c *offlineClient) NewWorkflowTemplateServiceClient() (workflowtemplate.WorkflowTemplateServiceClient, error) {
	return &errorTranslatingWorkflowTemplateServiceClient{OfflineWorkflowTemplateServiceClient{
		clusterWorkflowTemplateGetter:       c.clusterWorkflowTemplateGetter,
		namespacedWorkflowTemplateGetterMap: c.namespacedWorkflowTemplateGetterMap,
	}}, nil
}

func (c *offlineClient) NewClusterWorkflowTemplateServiceClient() (clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, error) {
	return &errorTranslatingWorkflowClusterTemplateServiceClient{OfflineClusterWorkflowTemplateServiceClient{
		clusterWorkflowTemplateGetter:       c.clusterWorkflowTemplateGetter,
		namespacedWorkflowTemplateGetterMap: c.namespacedWorkflowTemplateGetterMap,
	}}, nil
}

func (c *offlineClient) NewArchivedWorkflowServiceClient() (workflowarchivepkg.ArchivedWorkflowServiceClient, error) {
	return nil, NoArgoServerErr
}

func (c *offlineClient) NewInfoServiceClient() (infopkg.InfoServiceClient, error) {
	return nil, NoArgoServerErr
}

type offlineWorkflowTemplateNamespacedGetter struct {
	namespace         string
	workflowTemplates map[string]*wfv1.WorkflowTemplate
}

func (w offlineWorkflowTemplateNamespacedGetter) Get(name string) (*wfv1.WorkflowTemplate, error) {
	if v, ok := w.workflowTemplates[name]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("couldn't find workflow template %q in namespace %q", name, w.namespace)
}

type offlineClusterWorkflowTemplateGetter struct {
	clusterWorkflowTemplates map[string]*wfv1.ClusterWorkflowTemplate
}

func (o offlineClusterWorkflowTemplateGetter) Get(name string) (*wfv1.ClusterWorkflowTemplate, error) {
	if v, ok := o.clusterWorkflowTemplates[name]; ok {
		return v, nil
	}

	return nil, fmt.Errorf("couldn't find cluster workflow template %q", name)
}
