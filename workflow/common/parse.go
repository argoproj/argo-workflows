package common

import (
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kjson "sigs.k8s.io/json"
	"sigs.k8s.io/yaml"

	wf "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	jsonpkg "github.com/argoproj/argo-workflows/v3/util/json"
)

var yamlSeparator = regexp.MustCompile(`\n---`)

type ParseResult struct {
	Object metav1.Object
	Err    error
}

func ParseObjects(body []byte, strict bool) []ParseResult {
	var res []ParseResult
	if jsonpkg.IsJSON(body) {
		un := &unstructured.Unstructured{}
		err := jsonpkg.Unmarshal(body, un)
		if un.GetKind() != "" && err != nil {
			// only return an error if this is a kubernetes object, otherwise, ignore
			return append(res, ParseResult{nil, err})
		}
		v, err := toWorkflowTypeJSON(body, un.GetKind(), strict)
		return append(res, ParseResult{v, err})
	}

	for i, text := range yamlSeparator.Split(string(body), -1) {
		if strings.TrimSpace(text) == "" {
			continue
		}
		un := &unstructured.Unstructured{}
		err := yaml.Unmarshal([]byte(text), un)
		if err != nil {
			// Only return an error if this is a kubernetes object, otherwise, print the error
			if un.GetKind() != "" {
				res = append(res, ParseResult{nil, err})
			} else {
				log.Errorf("yaml file at index %d is not valid: %s", i, err)
			}
			continue
		}
		v, err := toWorkflowTypeYAML([]byte(text), un.GetKind(), strict)
		if v != nil {
			// only append when this is a Kubernetes object
			res = append(res, ParseResult{v, err})
		}
	}
	return res
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
	case wf.WorkflowTaskSetKind:
		return &wfv1.WorkflowTaskSet{}
	default:
		return &metav1.ObjectMeta{}
	}
}

func toWorkflowTypeYAML(body []byte, kind string, strict bool) (metav1.Object, error) {
	var json []byte
	var err error

	if strict {
		json, err = yaml.YAMLToJSONStrict(body)
	} else {
		json, err = yaml.YAMLToJSON(body)
	}
	if err != nil {
		return nil, err
	}

	return toWorkflowTypeJSON(json, kind, strict)
}

func toWorkflowTypeJSON(body []byte, kind string, strict bool) (metav1.Object, error) {
	v := objectForKind(kind)
	if strict {
		var strictErrs []error
		strictJSONErrs, err := kjson.UnmarshalStrict(body, v)
		if err != nil {
			// fatal decoding error, not due to strictness
			return v, err
		}
		strictErrs = append(strictErrs, strictJSONErrs...)

		if len(strictErrs) > 0 {
			// return the successfully decoded object along with the strict errors
			return v, runtime.NewStrictDecodingError(strictErrs)
		}
		return v, err
	}

	return v, jsonpkg.Unmarshal(body, v)
}

// SplitWorkflowYAMLFile is a helper to split a body into multiple workflow objects
func SplitWorkflowYAMLFile(body []byte, strict bool) ([]wfv1.Workflow, error) {
	manifests := make([]wfv1.Workflow, 0)
	for _, res := range ParseObjects(body, strict) {
		obj, err := res.Object, res.Err
		v, ok := obj.(*wfv1.Workflow)
		if !ok {
			log.Warnf("%s is not of kind Workflow. Ignoring...", obj.GetName())
			continue
		}
		if err != nil { // only returns parsing errors for workflow types
			return nil, err
		}
		manifests = append(manifests, *v)
	}
	return manifests, nil
}

// SplitWorkflowTemplateYAMLFile is a helper to split a body into multiple workflow template objects
func SplitWorkflowTemplateYAMLFile(body []byte, strict bool) ([]wfv1.WorkflowTemplate, error) {
	manifests := make([]wfv1.WorkflowTemplate, 0)
	for _, res := range ParseObjects(body, strict) {
		obj, err := res.Object, res.Err
		v, ok := obj.(*wfv1.WorkflowTemplate)
		if !ok {
			log.Warnf("%s is not of kind WorkflowTemplate. Ignoring...", obj.GetName())
			continue
		}
		if err != nil { // only returns parsing errors for template types
			return nil, err
		}
		manifests = append(manifests, *v)
	}
	return manifests, nil
}

// SplitCronWorkflowYAMLFile is a helper to split a body into multiple workflow template objects
func SplitCronWorkflowYAMLFile(body []byte, strict bool) ([]wfv1.CronWorkflow, error) {
	manifests := make([]wfv1.CronWorkflow, 0)
	for _, res := range ParseObjects(body, strict) {
		obj, err := res.Object, res.Err
		v, ok := obj.(*wfv1.CronWorkflow)
		if !ok {
			log.Warnf("%s is not of kind CronWorkflow. Ignoring...", obj.GetName())
			continue
		}
		if err != nil { // only returns parsing errors for cron types
			return nil, err
		}
		manifests = append(manifests, *v)
	}
	return manifests, nil
}

// SplitClusterWorkflowTemplateYAMLFile is a helper to split a body into multiple cluster workflow template objects
func SplitClusterWorkflowTemplateYAMLFile(body []byte, strict bool) ([]wfv1.ClusterWorkflowTemplate, error) {
	manifests := make([]wfv1.ClusterWorkflowTemplate, 0)
	for _, res := range ParseObjects(body, strict) {
		obj, err := res.Object, res.Err
		v, ok := obj.(*wfv1.ClusterWorkflowTemplate)
		if !ok {
			log.Warnf("%s is not of kind ClusterWorkflowTemplate. Ignoring...", obj.GetName())
			continue
		}
		if err != nil { // only returns parsing errors for cwft types
			return nil, err
		}
		manifests = append(manifests, *v)
	}
	return manifests, nil
}
