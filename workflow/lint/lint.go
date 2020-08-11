package lint

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo/pkg/apiclient"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

func Lint(ctx context.Context, apiClient apiclient.Client, defaultNamespace string, files []string, strict bool) {
	clusterWorkflowTemplateClient := apiClient.NewClusterWorkflowTemplateServiceClient()
	cronWorkflowsClient := apiClient.NewCronWorkflowServiceClient()
	workflowsClient := apiClient.NewWorkflowServiceClient()
	workflowTemplatesClient := apiClient.NewWorkflowTemplateServiceClient()
	lintData := func(data []byte) error {
		objects, err := common.ParseObjects(data, strict)
		if err != nil {
			return err
		}
		for _, obj := range objects {
			var err error
			// we should prefer the object's namespace
			namespace := obj.GetNamespace()
			if namespace == "" {
				namespace = defaultNamespace
			}
			switch v := obj.(type) {
			case *wfv1.ClusterWorkflowTemplate:
				_, err = clusterWorkflowTemplateClient.LintClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest{Template: v})
			case *wfv1.CronWorkflow:
				_, err = cronWorkflowsClient.LintCronWorkflow(ctx, &cronworkflowpkg.LintCronWorkflowRequest{Namespace: namespace, CronWorkflow: v})
			case *wfv1.Workflow:
				_, err = workflowsClient.LintWorkflow(ctx, &workflowpkg.WorkflowLintRequest{Namespace: namespace, Workflow: v})
			case *wfv1.WorkflowTemplate:
				_, err = workflowTemplatesClient.LintWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateLintRequest{Namespace: namespace, Template: v})
			default:
				// silently ignore unknown kinds
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
	invalid := false
	for _, file := range files {
		_ = filepath.Walk(file, func(file string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			switch filepath.Ext(info.Name()) {
			// empty string allows us to lint `/dev/stdin`
			case ".yaml", ".yml", ".json", "":
				data, err := ioutil.ReadFile(file)
				if err != nil {
					log.Errorf("%s: %s", file, err)
					invalid = true
					return nil
				}
				err = lintData(data)
				if err != nil {
					log.Errorf("%s: %s", file, err)
					invalid = true
					return nil
				}
				fmt.Printf("%s is valid\n", file)
			default:
				log.Warnf("%s: not .yaml, .yml, or .json", file)
			}
			return nil
		})
	}
	if invalid {
		log.Fatalf("Errors encountered in validation")
	}
	fmt.Printf("Manifests validated\n")
}
