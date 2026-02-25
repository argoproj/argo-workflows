package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
)

// SchemeGroupVersion is group version used to register these objects
var (
	SchemeGroupVersion             = schema.GroupVersion{Group: workflow.Group, Version: "v1alpha1"}
	WorkflowSchemaGroupVersionKind = schema.GroupVersionKind{Group: workflow.Group, Version: "v1alpha1", Kind: workflow.WorkflowKind}
)

// Kind takes an unqualified kind and returns back a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a Group-qualified GroupResource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Workflow{},
		&WorkflowList{},
		&WorkflowEventBinding{},
		&WorkflowEventBindingList{},
		&WorkflowTemplate{},
		&WorkflowTemplateList{},
		&CronWorkflow{},
		&CronWorkflowList{},
		&ClusterWorkflowTemplate{},
		&ClusterWorkflowTemplateList{},
		&WorkflowTaskSet{},
		&WorkflowTaskSetList{},
		&WorkflowArtifactGCTask{},
		&WorkflowArtifactGCTaskList{},
		&WorkflowTaskResult{},
		&WorkflowTaskResultList{},
		&WorkflowArtifactGCTask{},
		&WorkflowArtifactGCTaskList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
