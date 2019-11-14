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
	Status            CronWorkflowStatus  `json:"status" protobuf:"bytes,3,opt,name=status"`
	Options           CronWorkflowOptions `json:"options" protobuf:"bytes,4,opt,name=options"`
}

// CronWorkflowList is list of CronWorkflow resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CronWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []CronWorkflow `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type CronWorkflowStatus struct {
	// LastScheduleTime is the last time the CronWorkflow was scheduled
	LastScheduledTime *metav1.Time `json:"lastScheduledTime,omitempty" protobuf:"bytes,1,opt,name=lastScheduledTime"`
}

// CronWorkflowOptions is the schedule of when to run CronWorkflows
type CronWorkflowOptions struct {
	// Schedule is a schedule to run the Workflow in Cron format
	Schedule string `json:"schedule" protobuf:"bytes,1,opt,name=schedule"`
	// RuntimeNamespace is the namespace where the CronWorkflow will run
	RuntimeNamespace string `json:"runtimeNamespace" protobuf:"bytes,2,opt,name=runtimeNamespace"`
	// RuntimeGenerateName is the name generator the Workflow will be run with. Mutually exclusive with RuntimeName
	RuntimeGenerateName string `json:"runtimeGenerateName" protobuf:"bytes,3,opt,name=runtimeGenerateName"`
	// ConcurrencyPolicy is the K8s-style concurrency policy that will be used
	ConcurrencyPolicy v2alpha1.ConcurrencyPolicy `json:"concurrencyPolicy,omitempty" protobuf:"bytes,4,opt,name=concurrencyPolicy,casttype=ConcurrencyPolicy"`
	// Suspend is a flag that will stop new CronWorkflows from running if set to true
	Suspend bool `json:"suspend,omitempty" protobuf:"varint,5,opt,name=suspend"`
	// StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after its
	// original scheduled time if it is missed.
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty" protobuf:"varint,6,opt,name=startingDeadlineSeconds"`
	// SuccessfulJobsHistoryLimit is the K8s-style number of successful jobs that will be persisted
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty" protobuf:"varint,7,opt,name=successfulJobsHistoryLimit"`
	// FailedJobsHistoryLimit is the K8s-style number of failed jobs that will be persisted
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty" protobuf:"varint,8,opt,name=failedJobsHistoryLimit"`
}
