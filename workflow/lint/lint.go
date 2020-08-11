package lint

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo/pkg/apiclient"
	"github.com/argoproj/argo/pkg/apiclient/clusterworkflowtemplate"
	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

var AllKinds = map[string]bool{
	"ClusterWorkflowTemplate": true,
	"CronWorkflow":            true,
	"Workflow":                true,
	"WorkflowEventBinding":    true,
	"WorkflowTemplate":        true,
}

func OneKind(kind string) map[string]bool {
	return map[string]bool{kind: true}
}

func Lint(ctx context.Context, apiClient apiclient.Client, defaultNamespace string, files []string, strict bool, kinds map[string]bool) {
	clusterWorkflowTemplateClient := apiClient.NewClusterWorkflowTemplateServiceClient()
	cronWorkflowsClient := apiClient.NewCronWorkflowServiceClient()
	workflowsClient := apiClient.NewWorkflowServiceClient()
	workflowTemplatesClient := apiClient.NewWorkflowTemplateServiceClient()
	// if we did not lint anything at all - that is an error
	lintData := func(data []byte) (bool, error) {
		objects, err := common.ParseObjects(data, strict)
		if err != nil {
			return false, err
		}
		linted := false
		for _, obj := range objects {
			var err error
			// we should prefer the object's namespace
			namespace := obj.GetNamespace()
			if namespace == "" {
				namespace = defaultNamespace
			}
			// behaviour here:
			// - if the kind in a workflow kind, then we either ignore (if we're only linting specifically all kinds)
			// - otherwise we ignore the resource completely
			switch v := obj.(type) {
			case *wfv1.ClusterWorkflowTemplate:
				if kinds["ClusterWorkflowTemplate"] {
					linted = true
					_, err = clusterWorkflowTemplateClient.LintClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest{Template: v})
				}
			case *wfv1.CronWorkflow:
				if kinds["CronWorkflow"] {
					linted = true
					_, err = cronWorkflowsClient.LintCronWorkflow(ctx, &cronworkflowpkg.LintCronWorkflowRequest{Namespace: namespace, CronWorkflow: v})
				}
			case *wfv1.Workflow:
				if kinds["Workflow"] {
					linted = true
					_, err = workflowsClient.LintWorkflow(ctx, &workflowpkg.WorkflowLintRequest{Namespace: namespace, Workflow: v})
				}
			case *wfv1.WorkflowEventBinding:
				// noop
			case *wfv1.WorkflowTemplate:
				if kinds["WorkflowTemplate"] {
					linted = true
					_, err = workflowTemplatesClient.LintWorkflowTemplate(ctx, &workflowtemplatepkg.WorkflowTemplateLintRequest{Namespace: namespace, Template: v})
				}
			default:
				// silently ignore unknown kinds
			}
			if err != nil {
				return linted, err
			}
		}
		return linted, nil
	}
	// if we had any error - that is an error
	anyErrors := false
	// if we did not lint anything - that is an error
	anythingLinted := false
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
					anyErrors = true
					return nil
				}
				linted, err := lintData(data)
				if err != nil {
					log.Errorf("%s: %s", file, err)
					anyErrors = true
					return nil
				}
				if !linted {
					fmt.Printf("%s: ignored\n", file)
				} else {
					anythingLinted = true
					fmt.Printf("%s is valid\n", file)
				}
			default:
				log.Warnf("%s: not .yaml, .yml, or .json", file)
			}
			return nil
		})
	}
	if anyErrors {
		log.Fatalf("Errors encountered in validation")
	}
	if !anythingLinted {
		log.Fatalf("Error in %s: there was nothing to validate", strings.Join(files, ","))
	}
	fmt.Printf("Manifests validated\n")
}
