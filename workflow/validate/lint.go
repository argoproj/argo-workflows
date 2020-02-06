package validate

import (
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
func LintWorkflowTemplateDir(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, dirPath string, strict bool) error {
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
		return LintWorkflowTemplateFile(wftmplGetter, path, strict)
	}
	return filepath.Walk(dirPath, walkFunc)
}

// LintWorkflowTemplateFile lints a json file, or multiple workflow template manifest in a single yaml file. Ignores
// non-workflow template manifests
func LintWorkflowTemplateFile(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, filePath string, strict bool) error {
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
		err = ValidateWorkflowTemplate(wftmplGetter, &wftmpl)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "%s: %s", filePath, err.Error())
		}
	}
	return nil
}

// LintCronWorkflowDir validates all cron workflow manifests in a directory. Ignores non-workflow template manifests
func LintCronWorkflowDir(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, dirPath string, strict bool) error {
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
		return LintCronWorkflowFile(wftmplGetter, path, strict)
	}
	return filepath.Walk(dirPath, walkFunc)
}

// LintCronWorkflowFile lints a json file, or multiple cron workflow manifest in a single yaml file. Ignores
// non-cron workflow manifests
func LintCronWorkflowFile(wftmplGetter templateresolution.WorkflowTemplateNamespacedGetter, filePath string, strict bool) error {
	body, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "Can't read from file: %s, err: %v", filePath, err)
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
				// If we get here, it was a k8s manifest which was not of type 'Workflow'
				// We ignore these since we only care about validating Workflow manifests.
				return nil
			}
		}
	} else {
		cronWorkflows, err = common.SplitCronWorkflowYAMLFile(body, strict)
	}
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "%s failed to parse: %v", filePath, err)
	}
	for _, cronWf := range cronWorkflows {
		err = ValidateCronWorkflow(wftmplGetter, &cronWf)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "%s: %s", filePath, err.Error())
		}
	}
	return nil
}
