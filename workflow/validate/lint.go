package validate

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/argoproj/pkg/json"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

func ParseWfFromFile(filePath string, strict bool) ([]wfv1.Workflow, error) {
	body, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "Can't read from file: %s, err: %v", filePath, err)
	}
	var workflows []wfv1.Workflow
	if json.IsJSON(body) {
		var wf wfv1.Workflow
		if strict {
			err = json.UnmarshalStrict(body, &wf)
		} else {
			err = json.Unmarshal(body, &wf)
		}
		if err == nil {
			workflows = []wfv1.Workflow{wf}
		} else {
			if wf.Kind != "" && wf.Kind != workflow.WorkflowKind {
				// If we get here, it was a k8s manifest which was not of type 'Workflow'
				// We ignore these since we only care about validating Workflow manifests.
				return nil, nil
			}
		}
	} else {
		workflows, err = common.SplitWorkflowYAMLFile(body, strict)
	}
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "%s failed to parse: %v", filePath, err)
	}
	return workflows, nil
}

func ParseWfTmplFromFile(filePath string, strict bool) ([]wfv1.WorkflowTemplate, error) {
	body, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "Can't read from file: %s, err: %v", filePath, err)
	}
	var workflowTmpls []wfv1.WorkflowTemplate
	if json.IsJSON(body) {
		var wfTmpl wfv1.WorkflowTemplate
		if strict {
			err = json.UnmarshalStrict(body, &wfTmpl)
		} else {
			err = json.Unmarshal(body, &wfTmpl)
		}
		if err == nil {
			workflowTmpls = []wfv1.WorkflowTemplate{wfTmpl}
		} else {
			if wfTmpl.Kind != "" && wfTmpl.Kind != workflow.WorkflowTemplateKind {
				// If we get here, it was a k8s manifest which was not of type 'Workflow'
				// We ignore these since we only care about validating Workflow manifests.
				return nil, nil
			}
		}
	} else {
		workflowTmpls, err = common.SplitWorkflowTemplateYAMLFile(body, strict)
	}
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "%s failed to parse: %v", filePath, err)
	}
	return workflowTmpls, nil
}

func ParseCronWorkflowsFromFile(filePath string, strict bool) ([]wfv1.CronWorkflow, error) {
	body, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Can't read from file: %s, err: %v", filePath, err)
	}
	var cronWorkflows []wfv1.CronWorkflow
	if json.IsJSON(body) {
		var cronWf wfv1.CronWorkflow
		if strict {
			err = json.UnmarshalStrict(body, &cronWf)
		} else {
			err = json.Unmarshal(body, &cronWf)
		}
		if err == nil {
			cronWorkflows = []wfv1.CronWorkflow{cronWf}
		} else {
			if cronWf.Kind != "" && cronWf.Kind != workflow.CronWorkflowKind {
				// If we get here, it was a k8s manifest which was not of type 'CronWorkflow'
				// We ignore these since we only care about validating cron workflow manifests.
				return nil, nil
			}
		}
	} else {
		cronWorkflows, err = common.SplitCronWorkflowYAMLFile(body, strict)
	}
	if err != nil {
		return nil, fmt.Errorf("%s failed to parse: %v", filePath, err)
	}
	return cronWorkflows, nil
}

func ParseCWfTmplFromFile(filePath string, strict bool) ([]wfv1.ClusterWorkflowTemplate, error) {
	body, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "Can't read from file: %s, err: %v", filePath, err)
	}
	var clusterWorkflowTmpls []wfv1.ClusterWorkflowTemplate
	if json.IsJSON(body) {
		var cwfTmpl wfv1.ClusterWorkflowTemplate
		if strict {
			err = json.UnmarshalStrict(body, &cwfTmpl)
		} else {
			err = json.Unmarshal(body, &cwfTmpl)
		}
		if err == nil {
			clusterWorkflowTmpls = []wfv1.ClusterWorkflowTemplate{cwfTmpl}
		} else {
			if cwfTmpl.Kind != "" && cwfTmpl.Kind != workflow.ClusterWorkflowTemplateKind {
				// If we get here, it was a k8s manifest which was not of type 'Workflow'
				// We ignore these since we only care about validating Workflow manifests.
				return nil, nil
			}
		}
	} else {
		clusterWorkflowTmpls, err = common.SplitClusterWorkflowTemplateYAMLFile(body, strict)
	}
	if err != nil {
		return nil, errors.Errorf(errors.CodeBadRequest, "%s failed to parse: %v", filePath, err)
	}
	return clusterWorkflowTmpls, nil
}

// Wrapper for collection of all Argo resources parsed from a file / set of files / directory.
type ArgoResources struct {
	Workflows                []wfv1.Workflow
	WorkflowTemplates        []wfv1.WorkflowTemplate
	CronWorkflows            []wfv1.CronWorkflow
	ClusterWorkflowTemplates []wfv1.ClusterWorkflowTemplate
}

func ParseResourcesFromFiles(fileNames []string, strict bool) (*ArgoResources, error) {
	allWorkflows := make([]wfv1.Workflow, 0)
	allWorkflowTemplates := make([]wfv1.WorkflowTemplate, 0)
	allCronWorkflows := make([]wfv1.CronWorkflow, 0)
	allClusterWorkflowTemplates := make([]wfv1.ClusterWorkflowTemplate, 0)

	// Try parsing every type of Argo resource from the file.
	parseResources := func(fileName string) error {
		workflows, err := ParseWfFromFile(fileName, strict)
		if err != nil {
			return err
		}
		if workflows != nil {
			allWorkflows = append(allWorkflows, workflows...)
		}
		templates, err := ParseWfTmplFromFile(fileName, strict)
		if err != nil {
			return err
		}
		if templates != nil {
			allWorkflowTemplates = append(allWorkflowTemplates, templates...)
		}
		crons, err := ParseCronWorkflowsFromFile(fileName, strict)
		if err != nil {
			return err
		}
		if crons != nil {
			allCronWorkflows = append(allCronWorkflows, crons...)
		}
		clusterTemplates, err := ParseCWfTmplFromFile(fileName, strict)
		if err != nil {
			return err
		}
		if clusterTemplates != nil {
			allClusterWorkflowTemplates = append(allClusterWorkflowTemplates, clusterTemplates...)
		}
		return nil
	}

	for _, fileName := range fileNames {
		stat, err := os.Stat(fileName)
		if err != nil {
			log.Fatal(err)
		}
		if stat.IsDir() {
			err := filepath.Walk(fileName, func(path string, info os.FileInfo, err error) error {
				// If there was an error with the walk, return.
				if err != nil {
					return err
				}

				// Only try parsing file types that make sense.
				fileExt := filepath.Ext(info.Name())
				switch fileExt {
				case ".yaml", ".yml", ".json":
				default:
					return nil
				}

				return parseResources(fileName)
			})

			if err != nil {
				return nil, err
			}
		} else {
			err := parseResources(fileName)
			if err != nil {
				return nil, err
			}
		}
	}

	resources := ArgoResources{
		Workflows:                allWorkflows,
		WorkflowTemplates:        allWorkflowTemplates,
		CronWorkflows:            allCronWorkflows,
		ClusterWorkflowTemplates: allClusterWorkflowTemplates,
	}
	return &resources, nil
}
