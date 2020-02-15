package v1alpha1

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowTemplate is the definition of a workflow template resource
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowTemplateSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
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

var _ TemplateGetter = &WorkflowTemplate{}

// WorkflowTemplateSpec is a spec of WorkflowTemplate.
type WorkflowTemplateSpec struct {
	// Templates is a list of workflow templates.
	Templates []Template `json:"templates" protobuf:"bytes,1,rep,name=templates"`
	// Arguments hold arguments to the template.
	Arguments Arguments `json:"arguments,omitempty" protobuf:"bytes,3,opt,name=arguments"`
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

// GetTemplateScope returns the template scope of workflow template.
func (wftmpl *WorkflowTemplate) GetTemplateScope() string {
	return wftmpl.Name
}
