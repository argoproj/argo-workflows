package v1alpha1

import (
	"strings"

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
	// StopStrategy defines if the cron workflow will stop being triggered once a certain condition has been reached, involving a number of runs of the workflow
	StopStrategy *StopStrategy `json:"stopStrategy,omitempty" protobuf:"bytes,10,opt,name=stopStrategy"`
	// Schedules is a list of schedules to run the Workflow in Cron format
	Schedules []string `json:"schedules,omitempty" protobuf:"bytes,11,opt,name=schedules"`
}

// StopStrategy defines if the cron workflow will stop being triggered once a certain condition has been reached, involving a number of runs of the workflow
type StopStrategy struct {
	// Condition defines a condition that stops scheduling workflows when evaluates to true. Use the
	// keywords `failed` or `succeeded` to access the number of failed or successful child workflows.
	Condition string `json:"condition" protobuf:"bytes,1,opt,name=condition"`
}

// CronWorkflowStatus is the status of a CronWorkflow
type CronWorkflowStatus struct {
	// Active is a list of active workflows stemming from this CronWorkflow
	Active []v1.ObjectReference `json:"active" protobuf:"bytes,1,rep,name=active"`
	// LastScheduleTime is the last time the CronWorkflow was scheduled
	LastScheduledTime *metav1.Time `json:"lastScheduledTime" protobuf:"bytes,2,opt,name=lastScheduledTime"`
	// Conditions is a list of conditions the CronWorkflow may have
	Conditions Conditions `json:"conditions" protobuf:"bytes,3,rep,name=conditions"`
	// Succeeded is a counter of how many times the child workflows had success
	Succeeded int64 `json:"succeeded" protobuf:"varint,4,rep,name=succeeded"`
	// Failed is a counter of how many times a child workflow terminated in failed or errored state
	Failed int64 `json:"failed" protobuf:"varint,5,rep,name=failed"`
	// Phase defines the cron workflow phase. It is changed to Stopped when the stopping condition is achieved which stops new CronWorkflows from running
	Phase CronWorkflowPhase `json:"phase" protobuf:"varint,6,rep,name=phase"`
}

type CronWorkflowPhase string

const (
	ActivePhase  CronWorkflowPhase = "Active"
	StoppedPhase CronWorkflowPhase = "Stopped"
)

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

func (c *CronWorkflow) SetSchedules(schedules []string) {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	var scheduleString strings.Builder
	for i, schedule := range schedules {
		scheduleString.WriteString(schedule)
		if i != len(schedules)-1 {
			scheduleString.WriteString(",")
		}
	}
	c.Annotations[annotationKeyLatestSchedule] = scheduleString.String()
}

func (c *CronWorkflow) GetLatestSchedule() string {
	return c.Annotations[annotationKeyLatestSchedule]
}

// GetScheduleString returns the schedule expression with timezone, if available. If multiple
// expressions are configured it returns a comma separated list of cron expressions
func (c *CronWorkflowSpec) GetScheduleString() string {
	var scheduleString string
	if c.Schedule != "" {
		scheduleString = c.withTimezone(c.Schedule)
	} else {
		var sb strings.Builder
		for i, schedule := range c.Schedules {
			sb.WriteString(c.withTimezone(schedule))
			if i != len(c.Schedules)-1 {
				sb.WriteString(",")
			}
		}
		scheduleString = sb.String()
	}
	return scheduleString
}

// GetSchedulesWithTimezone returns all schedules configured for the CronWorkflow with a timezone. It handles
// both Spec.Schedules and Spec.Schedule for backwards compatibility
func (c *CronWorkflowSpec) GetSchedulesWithTimezone() []string {
	return c.getSchedules(true)
}

// GetSchedules returns all schedules configured for the CronWorkflow. It handles both Spec.Schedules
// and Spec.Schedule for backwards compatibility
func (c *CronWorkflowSpec) GetSchedules() []string {
	return c.getSchedules(false)
}

func (c *CronWorkflowSpec) getSchedules(withTimezone bool) []string {
	var schedules []string
	if c.Schedule != "" {
		schedule := c.Schedule
		if withTimezone {
			schedule = c.withTimezone(c.Schedule)
		}
		schedules = append(schedules, schedule)
	} else {
		schedules = make([]string, len(c.Schedules))
		for i, schedule := range c.Schedules {
			if withTimezone {
				schedule = c.withTimezone(schedule)
			}
			schedules[i] = c.withTimezone(schedule)
		}
	}
	return schedules
}

func (c *CronWorkflowSpec) withTimezone(scheduleString string) string {
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
