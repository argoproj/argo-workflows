package v1alpha1

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowTemplate is the definition of a workflow template resource
// +genclient
// +genclient:noStatus
// +kubebuilder:resource:shortName=wftmpl
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type WorkflowTemplates []WorkflowTemplate

func (w WorkflowTemplates) Len() int {
	return len(w)
}

func (w WorkflowTemplates) Less(i, j int) bool {
	return strings.Compare(w[j].ObjectMeta.Name, w[i].ObjectMeta.Name) > 0
}

func (w WorkflowTemplates) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}

// WorkflowTemplateList is list of WorkflowTemplate resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           WorkflowTemplates `json:"items" protobuf:"bytes,2,rep,name=items"`
}

var _ TemplateHolder = &WorkflowTemplate{}

// GetTemplateByName retrieves a defined template by its name
func (wftmpl *WorkflowTemplate) GetTemplateByName(name string) *Template {
	for _, t := range wftmpl.Spec.Templates {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// GetResourceScope returns the template scope of workflow template.
func (wftmpl *WorkflowTemplate) GetResourceScope() ResourceScope {
	return ResourceScopeNamespaced
}

// GetPodMetadata returns the PodMetadata of workflow template.
func (wftmpl *WorkflowTemplate) GetPodMetadata() *Metadata {
	return wftmpl.Spec.PodMetadata
}

// GetWorkflowSpec returns the WorkflowSpec of workflow template.
func (wftmpl *WorkflowTemplate) GetWorkflowSpec() *WorkflowSpec {
	return &wftmpl.Spec
}
