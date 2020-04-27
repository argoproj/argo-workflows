package v1alpha1

import (
	"strings"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterWorkflowTemplate is the definition of a workflow template resource in cluster scope
// +genclient
// +genclient:noStatus
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterWorkflowTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowTemplateSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type ClusterWorkflowTemplates []ClusterWorkflowTemplate

func (w ClusterWorkflowTemplates) Len() int {
	return len(w)
}

func (w ClusterWorkflowTemplates) Less(i, j int) bool {
	return strings.Compare(w[j].ObjectMeta.Name, w[i].ObjectMeta.Name) > 0
}

func (w ClusterWorkflowTemplates) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

// ClusterWorkflowTemplateList is list of ClusterWorkflowTemplate resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterWorkflowTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           ClusterWorkflowTemplates `json:"items" protobuf:"bytes,2,rep,name=items"`
}

var _ TemplateHolder = &ClusterWorkflowTemplate{}

// GetTemplateByName retrieves a defined template by its name
func (cwftmpl *ClusterWorkflowTemplate) GetTemplateByName(name string) *Template {
	for _, t := range cwftmpl.Spec.Templates {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// GetResourceScope returns the template scope of workflow template.
func (cwftmpl *ClusterWorkflowTemplate) GetResourceScope() ResourceScope {
	return ResourceScopeCluster
}

// GetArguments returns the Arguments.
func (cwftmpl *ClusterWorkflowTemplate) GetArguments() Arguments {
	return cwftmpl.Spec.Arguments
}

// GetEntrypoint returns the Entrypoint.
func (cwftmpl *ClusterWorkflowTemplate) GetEntrypoint() string {
	return cwftmpl.Spec.Entrypoint
}

// GetVolumes returns the Volumes
func (cwftmpl *ClusterWorkflowTemplate) GetVolumes() []apiv1.Volume {
	return cwftmpl.Spec.Volumes
}

func (cwftmpl *ClusterWorkflowTemplate) GetTemplates() []Template {
	return cwftmpl.Spec.Templates
}

func (cwftmpl *ClusterWorkflowTemplate) GetSpec() WorkflowSpec {
	return cwftmpl.Spec.WorkflowSpec
}