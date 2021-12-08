package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
)

// CronWorkflow is the definition of a scheduled workflow resource
// +genclient
// +genclient:noStatus
// +kubebuilder:resource:shortName=cwf;cronwf
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CronWorkflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              CronWorkflowSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	Status            CronWorkflowStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// CronWorkflowList is list of CronWorkflow resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CronWorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []CronWorkflow `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type ConcurrencyPolicy string

const (
	AllowConcurrent   ConcurrencyPolicy = "Allow"
	ForbidConcurrent  ConcurrencyPolicy = "Forbid"
	ReplaceConcurrent ConcurrencyPolicy = "Replace"
)

const annotationKeyLatestSchedule = workflow.CronWorkflowFullName + "/last-used-schedule"

// CronWorkflowSpec is the specification of a CronWorkflow
type CronWorkflowSpec struct {
	// WorkflowSpec is the spec of the workflow to be run
	WorkflowSpec WorkflowSpec `json:"workflowSpec" protobuf:"bytes,1,opt,name=workflowSpec,casttype=WorkflowSpec"`
	// Schedule is a schedule to run the Workflow in Cron format
	Schedule string `json:"schedule" protobuf:"bytes,2,opt,name=schedule"`
	// ConcurrencyPolicy is the K8s-style concurrency policy that will be used
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy,omitempty" protobuf:"bytes,3,opt,name=concurrencyPolicy,casttype=ConcurrencyPolicy"`
	// Suspend is a flag that will stop new CronWorkflows from running if set to true
	Suspend bool `json:"suspend,omitempty" protobuf:"varint,4,opt,name=suspend"`
	// StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after its
	// original scheduled time if it is missed.
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty" protobuf:"varint,5,opt,name=startingDeadlineSeconds"`
	// SuccessfulJobsHistoryLimit is the number of successful jobs to be kept at a time
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty" protobuf:"varint,6,opt,name=successfulJobsHistoryLimit"`
	// FailedJobsHistoryLimit is the number of failed jobs to be kept at a time
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty" protobuf:"varint,7,opt,name=failedJobsHistoryLimit"`
	// Timezone is the timezone against which the cron schedule will be calculated, e.g. "Asia/Tokyo". Default is machine's local time.
	Timezone string `json:"timezone,omitempty" protobuf:"bytes,8,opt,name=timezone"`
	// WorkflowMetadata contains some metadata of the workflow to be run
	WorkflowMetadata *metav1.ObjectMeta `json:"workflowMetadata,omitempty" protobuf:"bytes,9,opt,name=workflowMeta"`
}

// CronWorkflowStatus is the status of a CronWorkflow
type CronWorkflowStatus struct {
	// Active is a list of active workflows stemming from this CronWorkflow
	Active []v1.ObjectReference `json:"active" protobuf:"bytes,1,rep,name=active"`
	// LastScheduleTime is the last time the CronWorkflow was scheduled
	LastScheduledTime *metav1.Time `json:"lastScheduledTime" protobuf:"bytes,2,opt,name=lastScheduledTime"`
	// Conditions is a list of conditions the CronWorkflow may have
	Conditions Conditions `json:"conditions" protobuf:"bytes,3,rep,name=conditions"`
}

func (c *CronWorkflow) IsUsingNewSchedule() bool {
	lastUsedSchedule, exists := c.Annotations[annotationKeyLatestSchedule]
	// If last-used-schedule does not exist, or if it does not match the current schedule then the CronWorkflow schedule
	// was just updated
	return !exists || lastUsedSchedule != c.Spec.GetScheduleString()
}

func (c *CronWorkflow) SetSchedule(schedule string) {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[annotationKeyLatestSchedule] = schedule
}

func (c *CronWorkflow) GetLatestSchedule() string {
	return c.Annotations[annotationKeyLatestSchedule]
}

func (c *CronWorkflowSpec) GetScheduleString() string {
	scheduleString := c.Schedule
	if c.Timezone != "" {
		scheduleString = "CRON_TZ=" + c.Timezone + " " + scheduleString
	}
	return scheduleString
}

func (c *CronWorkflowStatus) HasActiveUID(uid types.UID) bool {
	for _, ref := range c.Active {
		if uid == ref.UID {
			return true
		}
	}
	return false
}

const (
	// ConditionTypeSubmissionError signifies that there was an error when submitting the CronWorkflow as a Workflow
	ConditionTypeSubmissionError ConditionType = "SubmissionError"
)
