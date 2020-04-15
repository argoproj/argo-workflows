package validate

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/argoproj/pkg/json"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/templateresolution"
)

// LintWorkflowDir validates all workflow manifests in a directory. Ignores non-workflow manifests

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

// LintWorkflowTemplateDir validates all workflow manifests in a directory. Ignores non-workflow template manifests
func LintWorkflowTemplateDir(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, dirPath string, strict bool) error {
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		fileExt := filepath.Ext(info.Name())
		switch fileExt {
		case ".yaml", ".yml", ".json":
		default:
			return nil
		}
		return LintWorkflowTemplateFile(wftmplGetter, cwftmplGetter, path, strict)
	}
	return filepath.Walk(dirPath, walkFunc)
}

// LintWorkflowTemplateFile lints a json file, or multiple workflow template manifest in a single yaml file. Ignores
// non-workflow template manifests
func LintWorkflowTemplateFile(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, cwftmplGetter templateresolution.ClusterWorkflowTemplateGetter, filePath string, strict bool) error {
	body, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "Can't read from file: %s, err: %v", filePath, err)
	}
	var workflowTemplates []wfv1.WorkflowTemplate
	if json.IsJSON(body) {
		var wftmpl wfv1.WorkflowTemplate
		if strict {
			err = json.UnmarshalStrict(body, &wftmpl)
		} else {
			err = json.Unmarshal(body, &wftmpl)
		}
		if err == nil {
			workflowTemplates = []wfv1.WorkflowTemplate{wftmpl}
		} else {
			if wftmpl.Kind != "" && wftmpl.Kind != workflow.WorkflowTemplateKind {
				// If we get here, it was a k8s manifest which was not of type 'Workflow'
				// We ignore these since we only care about validating Workflow manifests.
				return nil
			}
		}
	} else {
		workflowTemplates, err = common.SplitWorkflowTemplateYAMLFile(body, strict)
	}
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "%s failed to parse: %v", filePath, err)
	}
	for _, wftmpl := range workflowTemplates {
		_, err = ValidateWorkflowTemplate(wftmplGetter, cwftmplGetter, &wftmpl)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "%s: %s", filePath, err.Error())
		}
	}
	return nil
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
