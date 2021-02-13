package common

import (
	"regexp"
	"strings"

	jsonpkg "github.com/argoproj/pkg/json"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var yamlSeparator = regexp.MustCompile(`\n---`)

func ParseObjects(body []byte, strict bool) ([]metav1.Object, error) {
	if jsonpkg.IsJSON(body) {
		un := &unstructured.Unstructured{}
		err := jsonpkg.Unmarshal(body, un)
		if un.GetKind() != "" && err != nil {
			// only return an error if this is a kubernetes object, otherwise, ignore
			return nil, err
		}
		v, err := toWorkflowTypeJSON(body, un.GetKind(), strict)
		if err != nil {
			return nil, err
		}
		return []metav1.Object{v}, nil
	}

	manifests := make([]metav1.Object, 0)
	for _, text := range yamlSeparator.Split(string(body), -1) {
		if strings.TrimSpace(text) == "" {
			continue
		}
		un := &unstructured.Unstructured{}
		err := yaml.Unmarshal([]byte(text), un)
		if un.GetKind() != "" && err != nil {
			// only return an error if this is a kubernetes object, otherwise, ignore
			return nil, err
		}
		v, err := toWorkflowTypeYAML([]byte(text), un.GetKind(), strict)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, v)
	}
	return manifests, nil
}

func objectForKind(kind string) metav1.Object {
	switch kind {
	case wf.CronWorkflowKind:
		return &wfv1.CronWorkflow{}
	case wf.ClusterWorkflowTemplateKind:
		return &wfv1.ClusterWorkflowTemplate{}
	case wf.WorkflowKind:
		return &wfv1.Workflow{}
	case wf.WorkflowEventBindingKind:
		return &wfv1.WorkflowEventBinding{}
	case wf.WorkflowTemplateKind:
		return &wfv1.WorkflowTemplate{}
	default:
		return &metav1.ObjectMeta{}
	}
}

func toWorkflowTypeYAML(body []byte, kind string, strict bool) (metav1.Object, error) {
	var opts []yaml.JSONOpt

	v := objectForKind(kind)
	if strict {
		opts = append(opts, yaml.DisallowUnknownFields)
	}

	return v, yaml.Unmarshal(body, v, opts...)
}

func toWorkflowTypeJSON(body []byte, kind string, strict bool) (metav1.Object, error) {
	v := objectForKind(kind)
	if strict {
		return v, jsonpkg.UnmarshalStrict(body, v)
	}

	return v, jsonpkg.Unmarshal(body, v)
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
