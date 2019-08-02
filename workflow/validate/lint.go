package validate

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/argoproj/pkg/json"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
)

// LintWorkflowDir validates all workflow manifests in a directory. Ignores non-workflow manifests
func LintWorkflowDir(wfClientset wfclientset.Interface, namespace, dirPath string, strict bool) error {
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
		return LintWorkflowFile(wfClientset, namespace, path, strict)
	}
	return filepath.Walk(dirPath, walkFunc)
}

// LintWorkflowFile lints a json file, or multiple workflow manifest in a single yaml file. Ignores
// non-workflow manifests
func LintWorkflowFile(wfClientset wfclientset.Interface, namespace, filePath string, strict bool) error {
	body, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "Can't read from file: %s, err: %v", filePath, err)
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
				return nil
			}
		}
	} else {
		workflows, err = common.SplitWorkflowYAMLFile(body, strict)
	}
	if err != nil {
		return errors.Errorf(errors.CodeBadRequest, "%s failed to parse: %v", filePath, err)
	}
	for _, wf := range workflows {
		err = ValidateWorkflow(wfClientset, namespace, &wf, ValidateOpts{Lint: true})
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "%s: %s", filePath, err.Error())
		}
	}
	return nil
}

// LintWorkflowTemplateDir validates all workflow manifests in a directory. Ignores non-workflow template manifests
func LintWorkflowTemplateDir(wfClientset wfclientset.Interface, namespace, dirPath string, strict bool) error {
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
		return LintWorkflowTemplateFile(wfClientset, namespace, path, strict)
	}
	return filepath.Walk(dirPath, walkFunc)
}

// LintWorkflowTemplateFile lints a json file, or multiple workflow template manifest in a single yaml file. Ignores
// non-workflow template manifests
func LintWorkflowTemplateFile(wfClientset wfclientset.Interface, namespace, filePath string, strict bool) error {
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
		err = ValidateWorkflowTemplate(wfClientset, namespace, &wftmpl)
		if err != nil {
			return errors.Errorf(errors.CodeBadRequest, "%s: %s", filePath, err.Error())
		}
	}
	return nil
}
