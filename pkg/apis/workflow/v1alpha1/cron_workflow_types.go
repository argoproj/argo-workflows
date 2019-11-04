package v1alpha1

import (
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

// CronWorkflowOptions is the schedule of when to run CronWorkflows
type CronWorkflowOptions struct {
	// CronSchedule is a schedule to run the Workflow in Cron format
	CronSchedule string `json:"cronSchedule" protobuf:"bytes,1,opt,name=cronSchedule"`
}
