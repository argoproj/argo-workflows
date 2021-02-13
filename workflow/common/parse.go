package common

import (
	"regexp"
	"strings"

	jsonpkg "github.com/argoproj/pkg/json"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var yamlSeparator = regexp.MustCompile(`\n---`)

func ParseObjects(body []byte, strict bool) ([]metav1.Object, error) {
	if jsonpkg.IsJSON(body) {
		un := &unstructured.Unstructured{}
		var err error
		if strict {
			err = jsonpkg.UnmarshalStrict(body, un)
		} else {
			err = jsonpkg.Unmarshal(body, un)
		}
		if un.GetKind() != "" && err != nil {
			// only return an error if this is a kubernetes object, otherwise, ignore
			return nil, err
		}
		v, err := toWorkflowType(un)
		if err != nil {
			return nil, err
		}
		return []metav1.Object{v}, nil
	}
	manifests := make([]metav1.Object, 0)
	var opts []yaml.JSONOpt
	if strict {
		opts = append(opts, yaml.DisallowUnknownFields)
	}
	for _, text := range yamlSeparator.Split(string(body), -1) {
		if strings.TrimSpace(text) == "" {
			continue
		}
		un := &unstructured.Unstructured{}
		err := yaml.Unmarshal([]byte(text), un, opts...)
		if un.GetKind() != "" && err != nil {
			// only return an error if this is a kubernetes object, otherwise, ignore
			return nil, err
		}
		v, err := toWorkflowType(un)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, v)
	}
	return manifests, nil
}

func toWorkflowType(un *unstructured.Unstructured) (metav1.Object, error) {
	var v metav1.Object
	var err error
	switch un.GetKind() {
	case "CronWorkflow":
		v = &wfv1.CronWorkflow{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	case "ClusterWorkflowTemplate":
		v = &wfv1.ClusterWorkflowTemplate{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	case "Workflow":
		v = &wfv1.Workflow{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	case "WorkflowEventBinding":
		v = &wfv1.WorkflowEventBinding{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	case "WorkflowTemplate":
		v = &wfv1.WorkflowTemplate{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	default:
		v = un
	}
	return v, err
}

// SplitWorkflowYAMLFile is a helper to split a body into multiple workflow objects
func SplitWorkflowYAMLFile(body []byte, strict bool) ([]wfv1.Workflow, error) {
	objects, err := ParseObjects(body, strict)
	if err != nil {
		return nil, err
	}
	manifests := make([]wfv1.Workflow, 0)
	for _, obj := range objects {
		v, ok := obj.(*wfv1.Workflow)
		if !ok {
			log.Warnf("%s is not of kind Workflow. Ignoring...", obj.GetName())
			continue
		}
		manifests = append(manifests, *v)
	}
	return manifests, nil
}

// SplitWorkflowTemplateYAMLFile is a helper to split a body into multiple workflow template objects
func SplitWorkflowTemplateYAMLFile(body []byte, strict bool) ([]wfv1.WorkflowTemplate, error) {
	objects, err := ParseObjects(body, strict)
	if err != nil {
		return nil, err
	}
	manifests := make([]wfv1.WorkflowTemplate, 0)
	for _, obj := range objects {
		v, ok := obj.(*wfv1.WorkflowTemplate)
		if !ok {
			log.Warnf("%s is not of kind WorkflowTemplate. Ignoring...", obj.GetName())
			continue
		}
		manifests = append(manifests, *v)
	}
	return manifests, nil
}

// SplitCronWorkflowYAMLFile is a helper to split a body into multiple workflow template objects
func SplitCronWorkflowYAMLFile(body []byte, strict bool) ([]wfv1.CronWorkflow, error) {
	objects, err := ParseObjects(body, strict)
	if err != nil {
		return nil, err
	}
	manifests := make([]wfv1.CronWorkflow, 0)
	for _, obj := range objects {
		v, ok := obj.(*wfv1.CronWorkflow)
		if !ok {
			log.Warnf("%s is not of kind CronWorkflow. Ignoring...", obj.GetName())
			continue
		}
		manifests = append(manifests, *v)
	}
	return manifests, nil
}

// SplitClusterWorkflowTemplateYAMLFile is a helper to split a body into multiple cluster workflow template objects
func SplitClusterWorkflowTemplateYAMLFile(body []byte, strict bool) ([]wfv1.ClusterWorkflowTemplate, error) {
	objects, err := ParseObjects(body, strict)
	if err != nil {
		return nil, err
	}
	manifests := make([]wfv1.ClusterWorkflowTemplate, 0)
	for _, obj := range objects {
		v, ok := obj.(*wfv1.ClusterWorkflowTemplate)
		if !ok {
			log.Warnf("%s is not of kind ClusterWorkflowTemplate. Ignoring...", obj.GetName())
			continue
		}
		manifests = append(manifests, *v)
	}
	return manifests, nil
}
