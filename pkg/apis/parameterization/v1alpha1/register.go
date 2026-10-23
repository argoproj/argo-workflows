package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName remains experimental until an upstream API group is approved.
const GroupName = "parameters.workflows.example.io"

var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}
var SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
var AddToScheme = SchemeBuilder.AddToScheme

func Kind(kind string) schema.GroupKind { return SchemeGroupVersion.WithKind(kind).GroupKind() }
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ParameterTemplate{}, &ParameterTemplateList{},
		&ParameterSet{}, &ParameterSetList{},
		&ParameterRevision{}, &ParameterRevisionList{},
		&WorkflowForm{}, &WorkflowFormList{},
		&WorkflowParameterBinding{}, &WorkflowParameterBindingList{},
		&WorkflowSubmission{}, &WorkflowSubmissionList{},
		&ActionDefinition{}, &ActionDefinitionList{},
		&ActionResultTemplate{}, &ActionResultTemplateList{},
		&WorkflowActionBinding{}, &WorkflowActionBindingList{},
		&ParameterRetry{}, &ParameterRetryList{},
		&ParameterNodeInput{}, &ParameterNodeInputList{},
		&ResourceOperation{}, &ResourceOperationList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
