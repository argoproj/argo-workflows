package v1alpha1

import (
	"k8s.io/api/batch/v2alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CronWorkflow is the definition of a scheduled workflow resource
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CronWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowSpec        `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Options           CronWorkflowOptions `json:"options" protobuf:"bytes,3,opt,name=options"`
}

// CronWorkflowList is list of CronWorkflow resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CronWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []CronWorkflow `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// TODO: Consider replacing this with the K8s API CronJobSpec. This spec is only available starting on v2alpha1.
// CronWorkflowOptions is the schedule of when to run CronWorkflows
type CronWorkflowOptions struct {
	// Schedule is a schedule to run the Workflow in Cron format
	Schedule string `json:"schedule" protobuf:"bytes,1,opt,name=schedule"`
	// RuntimeNamespace is the namespace where the CronWorkflow will run
	RuntimeNamespace string `json:"runtimeNamespace" protobuf:"bytes,2,opt,name=runtimeNamespace"`
	// RuntimeGenerateName is the name generator the Workflow will be run with. Mutually exclusive with RuntimeName
	RuntimeGenerateName string `json:"runtimeGenerateName" protobuf:"bytes,3,opt,name=runtimeGenerateName"`
	// ConcurrencyPolicy is the name generator the Workflow will be run with. Mutually exclusive with RuntimeName
	ConcurrencyPolicy v2alpha1.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty" protobuf:"bytes,4,opt,name=concurrencyPolicy,casttype=ConcurrencyPolicy"`
}
