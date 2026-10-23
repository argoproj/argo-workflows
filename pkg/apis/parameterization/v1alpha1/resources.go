package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ParameterTemplate is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ParameterTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ParameterTemplateSpec `json:"spec"`
	Status            DefinitionStatus      `json:"status,omitempty"`
}

// ParameterTemplateList is a list of ParameterTemplate resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ParameterTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ParameterTemplate `json:"items"`
}

// ParameterSet is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ParameterSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ParameterSetSpec `json:"spec"`
	Status            DefinitionStatus `json:"status,omitempty"`
}

// ParameterSetList is a list of ParameterSet resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ParameterSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ParameterSet `json:"items"`
}

// ParameterRevision is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ParameterRevision struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ParameterRevisionSpec `json:"spec"`
	Status            DefinitionStatus      `json:"status,omitempty"`
}

// ParameterRevisionList is a list of ParameterRevision resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ParameterRevisionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ParameterRevision `json:"items"`
}

// WorkflowForm is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowForm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WorkflowFormSpec `json:"spec"`
	Status            DefinitionStatus `json:"status,omitempty"`
}

// WorkflowFormList is a list of WorkflowForm resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowFormList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []WorkflowForm `json:"items"`
}

// WorkflowParameterBinding is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowParameterBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WorkflowParameterBindingSpec `json:"spec"`
	Status            DefinitionStatus             `json:"status,omitempty"`
}

// WorkflowParameterBindingList is a list of WorkflowParameterBinding resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowParameterBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []WorkflowParameterBinding `json:"items"`
}

// WorkflowSubmission is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowSubmission struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WorkflowSubmissionSpec `json:"spec"`
	Status            DefinitionStatus       `json:"status,omitempty"`
}

// WorkflowSubmissionList is a list of WorkflowSubmission resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowSubmissionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []WorkflowSubmission `json:"items"`
}

// ActionDefinition is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ActionDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ActionDefinitionSpec `json:"spec"`
	Status            DefinitionStatus     `json:"status,omitempty"`
}

// ActionDefinitionList is a list of ActionDefinition resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ActionDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ActionDefinition `json:"items"`
}

// ActionResultTemplate is a reusable public result source, schema and view.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ActionResultTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ActionResultTemplateSpec `json:"spec"`
	Status            DefinitionStatus         `json:"status,omitempty"`
}

// ActionResultTemplateList is a list of ActionResultTemplate resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ActionResultTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ActionResultTemplate `json:"items"`
}

// WorkflowActionBinding is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowActionBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              WorkflowActionBindingSpec `json:"spec"`
	Status            DefinitionStatus          `json:"status,omitempty"`
}

// WorkflowActionBindingList is a list of WorkflowActionBinding resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowActionBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []WorkflowActionBinding `json:"items"`
}

// ParameterRetry is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ParameterRetry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ParameterRetrySpec `json:"spec"`
	Status            DefinitionStatus   `json:"status,omitempty"`
}

// ParameterRetryList is a list of ParameterRetry resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ParameterRetryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ParameterRetry `json:"items"`
}

// ParameterNodeInput is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ParameterNodeInput struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ParameterNodeInputSpec `json:"spec"`
	Status            DefinitionStatus       `json:"status,omitempty"`
}

// ParameterNodeInputList is a list of ParameterNodeInput resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ParameterNodeInputList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ParameterNodeInput `json:"items"`
}

// ResourceOperation is a namespaced structured parameter API resource.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ResourceOperation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ResourceOperationSpec   `json:"spec"`
	Status            ResourceOperationStatus `json:"status,omitempty"`
}

// ResourceOperationList is a list of ResourceOperation resources.
// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ResourceOperationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ResourceOperation `json:"items"`
}
