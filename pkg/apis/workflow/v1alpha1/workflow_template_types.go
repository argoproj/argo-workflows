package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowTemplate is the definition of a workflow template resource
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkflowTemplateSpec `json:"spec"`
}

// WorkflowTemplateList is list of WorkflowTemplate resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []WorkflowTemplate `json:"items"`
}

var _ TemplateGetter = &WorkflowTemplate{}

// WorkflowTemplateSpec is a spec of WorkflowTemplate.
type WorkflowTemplateSpec struct {
	// Templates is a list of workflow templates.
	Templates []Template `json:"templates"`
	// Arguments hold arguments to the template.
	Arguments Arguments `json:"arguments,omitempty"`
}

// GetTemplateByName retrieves a defined template by its name
func (wftmpl *WorkflowTemplate) GetTemplateByName(name string) *Template {
	for _, t := range wftmpl.Spec.Templates {
		if t.Name == name {
			return &t
		}
	}
	return nil
}
