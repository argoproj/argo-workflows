package v1alpha1

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"reflect"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	policyv1beta "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TemplateType is the type of a template
type TemplateType string

// Possible template types
const (
	TemplateTypeContainer TemplateType = "Container"
	TemplateTypeSteps     TemplateType = "Steps"
	TemplateTypeScript    TemplateType = "Script"
	TemplateTypeResource  TemplateType = "Resource"
	TemplateTypeDAG       TemplateType = "DAG"
	TemplateTypeSuspend   TemplateType = "Suspend"
	TemplateTypeUnknown   TemplateType = "Unknown"
)

// NodePhase is a label for the condition of a node at the current time.
type NodePhase string

// Workflow and node statuses
const (
	NodePending   NodePhase = "Pending"
	NodeRunning   NodePhase = "Running"
	NodeSucceeded NodePhase = "Succeeded"
	NodeSkipped   NodePhase = "Skipped"
	NodeFailed    NodePhase = "Failed"
	NodeError     NodePhase = "Error"
)

// NodeType is the type of a node
type NodeType string

// Node types
const (
	NodeTypePod       NodeType = "Pod"
	NodeTypeSteps     NodeType = "Steps"
	NodeTypeStepGroup NodeType = "StepGroup"
	NodeTypeDAG       NodeType = "DAG"
	NodeTypeTaskGroup NodeType = "TaskGroup"
	NodeTypeRetry     NodeType = "Retry"
	NodeTypeSkipped   NodeType = "Skipped"
	NodeTypeSuspend   NodeType = "Suspend"
)

// PodGCStrategy is the strategy when to delete completed pods for GC.
type PodGCStrategy string

// PodGCStrategy
const (
	PodGCOnPodCompletion      PodGCStrategy = "OnPodCompletion"
	PodGCOnPodSuccess         PodGCStrategy = "OnPodSuccess"
	PodGCOnWorkflowCompletion PodGCStrategy = "OnWorkflowCompletion"
	PodGCOnWorkflowSuccess    PodGCStrategy = "OnWorkflowSuccess"
)

// Workflow is the definition of a workflow resource
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Spec              WorkflowSpec   `json:"spec" protobuf:"bytes,2,opt,name=spec "`
	Status            WorkflowStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// Workflows is a sort interface which sorts running jobs earlier before considering FinishedAt
type Workflows []Workflow

func (w Workflows) Len() int      { return len(w) }
func (w Workflows) Swap(i, j int) { w[i], w[j] = w[j], w[i] }
func (w Workflows) Less(i, j int) bool {
	iStart := w[i].ObjectMeta.CreationTimestamp
	iFinish := w[i].Status.FinishedAt
	jStart := w[j].ObjectMeta.CreationTimestamp
	jFinish := w[j].Status.FinishedAt
	if iFinish.IsZero() && jFinish.IsZero() {
		return !iStart.Before(&jStart)
	}
	if iFinish.IsZero() && !jFinish.IsZero() {
		return true
	}
	if !iFinish.IsZero() && jFinish.IsZero() {
		return false
	}
	return jFinish.Before(&iFinish)
}

// WorkflowList is list of Workflow resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           Workflows `json:"items" protobuf:"bytes,2,opt,name=items"`
}

var _ TemplateHolder = &Workflow{}

// TTLStrategy is the strategy for the time to live depending on if the workflow succeeded or failed
type TTLStrategy struct {
	// SecondsAfterCompletion is the number of seconds to live after completion
	SecondsAfterCompletion *int32 `json:"secondsAfterCompletion,omitempty" protobuf:"bytes,1,opt,name=secondsAfterCompletion"`
	// SecondsAfterSuccess is the number of seconds to live after success
	SecondsAfterSuccess *int32 `json:"secondsAfterSuccess,omitempty" protobuf:"bytes,2,opt,name=secondsAfterSuccess"`
	// SecondsAfterFailure is the number of seconds to live after failure
	SecondsAfterFailure *int32 `json:"secondsAfterFailure,omitempty" protobuf:"bytes,3,opt,name=secondsAfterFailure"`
}

// WorkflowSpec is the specification of a Workflow.
type WorkflowSpec struct {
	// Templates is a list of workflow templates used in a workflow
	// +patchStrategy=merge
	// +patchMergeKey=name
	Templates []Template `json:"templates" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,1,opt,name=templates"`

	// Entrypoint is a template reference to the starting point of the workflow.
	Entrypoint string `json:"entrypoint,omitempty" protobuf:"bytes,2,opt,name=entrypoint"`

	// Arguments contain the parameters and artifacts sent to the workflow entrypoint
	// Parameters are referencable globally using the 'workflow' variable prefix.
	// e.g. {{workflow.parameters.myparam}}
	Arguments Arguments `json:"arguments,omitempty" protobuf:"bytes,3,opt,name=arguments"`

	// ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as.
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,4,opt,name=serviceAccountName"`

	// AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods.
	// ServiceAccountName of ExecutorConfig must be specified if this value is false.
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty" protobuf:"varint,28,opt,name=automountServiceAccountToken"`

	// Executor holds configurations of executor containers of the workflow.
	Executor *ExecutorConfig `json:"executor,omitempty" protobuf:"bytes,29,opt,name=executor"`

	// Volumes is a list of volumes that can be mounted by containers in a workflow.
	// +patchStrategy=merge
	// +patchMergeKey=name
	Volumes []apiv1.Volume `json:"volumes,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,5,opt,name=volumes"`

	// VolumeClaimTemplates is a list of claims that containers are allowed to reference.
	// The Workflow controller will create the claims at the beginning of the workflow
	// and delete the claims upon completion of the workflow
	// +patchStrategy=merge
	// +patchMergeKey=name
	VolumeClaimTemplates []apiv1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,6,opt,name=volumeClaimTemplates"`

	// Parallelism limits the max total parallel pods that can execute at the same time in a workflow
	Parallelism *int64 `json:"parallelism,omitempty" protobuf:"bytes,7,opt,name=parallelism"`

	// ArtifactRepositoryRef specifies the configMap name and key containing the artifact repository config.
	ArtifactRepositoryRef *ArtifactRepositoryRef `json:"artifactRepositoryRef,omitempty" protobuf:"bytes,8,opt,name=artifactRepositoryRef"`

	// Suspend will suspend the workflow and prevent execution of any future steps in the workflow
	Suspend *bool `json:"suspend,omitempty" protobuf:"bytes,9,opt,name=suspend"`

	// NodeSelector is a selector which will result in all pods of the workflow
	// to be scheduled on the selected node(s). This is able to be overridden by
	// a nodeSelector specified in the template.
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,10,opt,name=nodeSelector"`

	// Affinity sets the scheduling constraints for all pods in the workflow.
	// Can be overridden by an affinity specified in the template
	Affinity *apiv1.Affinity `json:"affinity,omitempty" protobuf:"bytes,11,opt,name=affinity"`

	// Tolerations to apply to workflow pods.
	// +patchStrategy=merge
	// +patchMergeKey=key
	Tolerations []apiv1.Toleration `json:"tolerations,omitempty" patchStrategy:"merge" patchMergeKey:"key" protobuf:"bytes,12,opt,name=tolerations"`

	// ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images
	// in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets
	// can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet.
	// More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod
	// +patchStrategy=merge
	// +patchMergeKey=name
	ImagePullSecrets []apiv1.LocalObjectReference `json:"imagePullSecrets,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,13,opt,name=imagePullSecrets"`

	// Host networking requested for this workflow pod. Default to false.
	HostNetwork *bool `json:"hostNetwork,omitempty" protobuf:"bytes,14,opt,name=hostNetwork"`

	// Set DNS policy for the pod.
	// Defaults to "ClusterFirst".
	// Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'.
	// DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy.
	// To have DNS options set along with hostNetwork, you have to specify DNS policy
	// explicitly to 'ClusterFirstWithHostNet'.
	DNSPolicy *apiv1.DNSPolicy `json:"dnsPolicy,omitempty" protobuf:"bytes,15,opt,name=dnsPolicy"`

	// PodDNSConfig defines the DNS parameters of a pod in addition to
	// those generated from DNSPolicy.
	DNSConfig *apiv1.PodDNSConfig `json:"dnsConfig,omitempty" protobuf:"bytes,16,opt,name=dnsConfig"`

	// OnExit is a template reference which is invoked at the end of the
	// workflow, irrespective of the success, failure, or error of the
	// primary workflow.
	OnExit string `json:"onExit,omitempty" protobuf:"bytes,17,opt,name=onExit"`

	// TTLSecondsAfterFinished limits the lifetime of a Workflow that has finished execution
	// (Succeeded, Failed, Error). If this field is set, once the Workflow finishes, it will be
	// deleted after ttlSecondsAfterFinished expires. If this field is unset,
	// ttlSecondsAfterFinished will not expire. If this field is set to zero,
	// ttlSecondsAfterFinished expires immediately after the Workflow finishes.
	// DEPRECATED: Use TTLStrategy.SecondsAfterCompletion instead.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty" protobuf:"bytes,18,opt,name=ttlSecondsAfterFinished"`

	// TTLStrategy limits the lifetime of a Workflow that has finished execution depending on if it
	// Succeeded or Failed. If this struct is set, once the Workflow finishes, it will be
	// deleted after the time to live expires. If this field is unset,
	// the controller config map will hold the default values.
	TTLStrategy *TTLStrategy `json:"ttlStrategy,omitempty" protobuf:"bytes,30,opt,name=ttlStrategy"`

	// Optional duration in seconds relative to the workflow start time which the workflow is
	// allowed to run before the controller terminates the workflow. A value of zero is used to
	// terminate a Running workflow
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty" protobuf:"bytes,19,opt,name=activeDeadlineSeconds"`

	// Priority is used if controller is configured to process limited number of workflows in parallel. Workflows with higher priority are processed first.
	Priority *int32 `json:"priority,omitempty" protobuf:"bytes,20,opt,name=priority"`

	// Set scheduler name for all pods.
	// Will be overridden if container/script template's scheduler name is set.
	// Default scheduler will be used if neither specified.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty" protobuf:"bytes,21,opt,name=schedulerName"`

	// PodGC describes the strategy to use when to deleting completed pods
	PodGC *PodGC `json:"podGC,omitempty" protobuf:"bytes,22,opt,name=podGC"`

	// PriorityClassName to apply to workflow pods.
	PodPriorityClassName string `json:"podPriorityClassName,omitempty" protobuf:"bytes,23,opt,name=podPriorityClassName"`

	// Priority to apply to workflow pods.
	PodPriority *int32 `json:"podPriority,omitempty" protobuf:"bytes,24,opt,name=podPriority"`

	// +patchStrategy=merge
	// +patchMergeKey=ip
	HostAliases []apiv1.HostAlias `json:"hostAliases,omitempty" patchStrategy:"merge" patchMergeKey:"ip" protobuf:"bytes,25,opt,name=hostAliases"`

	// SecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	// +optional
	SecurityContext *apiv1.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,26,opt,name=securityContext"`

	// PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of
	// container fields which are not strings (e.g. resource limits).
	PodSpecPatch string `json:"podSpecPatch,omitempty" protobuf:"bytes,27,opt,name=podSpecPatch"`

	//PodDisruptionBudget holds the number of concurrent disruptions that you allow for Workflow's Pods.
	//Controller will automatically add the selector with workflow name, if selector is empty.
	//Optional: Defaults to empty.
	// +optional
	PodDisruptionBudget *policyv1beta.PodDisruptionBudgetSpec `json:"podDisruptionBudget,omitempty" protobuf:"bytes,31,opt,name=podDisruptionBudget"`

	// Metrics are a list of metrics emitted from this Workflow
	Metrics *Metrics `json:"metrics,omitempty" protobuf:"bytes,32,opt,name=metrics"`

	// Shutdown will shutdown the workflow according to its ShutdownStrategy
	Shutdown ShutdownStrategy `json:"shutdown,omitempty" protobuf:"bytes,33,opt,name=shutdown,casttype=ShutdownStrategy"`

	// WorkflowTemplateRef holds a reference to a WorkflowTemplate for execution
	WorkflowTemplateRef *WorkflowTemplateRef `json:"workflowTemplateRef,omitempty" protobuf:"bytes,34,opt,name=workflowTemplateRef"`
}

type ShutdownStrategy string

const (
	ShutdownStrategyTerminate ShutdownStrategy = "Terminate"
	ShutdownStrategyStop      ShutdownStrategy = "Stop"
)

func (s ShutdownStrategy) ShouldExecute(isOnExitPod bool) bool {
	switch s {
	case ShutdownStrategyTerminate:
		return false
	case ShutdownStrategyStop:
		return isOnExitPod
	default:
		return true
	}
}

type ParallelSteps struct {
	Steps []WorkflowStep `protobuf:"bytes,1,rep,name=steps"`
}

// WorkflowStep is an anonymous list inside of ParallelSteps (i.e. it does not have a key), so it needs its own
// custom Unmarshaller
func (p *ParallelSteps) UnmarshalJSON(value []byte) error {
	// Since we are writing a custom unmarshaller, we have to enforce the "DisallowUnknownFields" requirement manually.

	// First, get a generic representation of the contents
	var candidate []map[string]interface{}
	err := json.Unmarshal(value, &candidate)
	if err != nil {
		return err
	}

	// Generate a list of all the available JSON fields of the WorkflowStep struct
	availableFields := map[string]bool{}
	reflectType := reflect.TypeOf(WorkflowStep{})
	for i := 0; i < reflectType.NumField(); i++ {
		cleanString := strings.ReplaceAll(reflectType.Field(i).Tag.Get("json"), ",omitempty", "")
		availableFields[cleanString] = true
	}

	// Enforce that no unknown fields are present
	for _, step := range candidate {
		for key := range step {
			if _, ok := availableFields[key]; !ok {
				return fmt.Errorf(`json: unknown field "%s"`, key)
			}
		}
	}

	// Finally, attempt to fully unmarshal the struct
	err = json.Unmarshal(value, &p.Steps)
	if err != nil {
		return err
	}
	return nil
}

func (p *ParallelSteps) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Steps)
}

func (wfs *WorkflowSpec) HasPodSpecPatch() bool {
	return wfs.PodSpecPatch != ""
}

// Template is a reusable and composable unit of execution in a workflow
type Template struct {
	// Name is the name of the template
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Template is the name of the template which is used as the base of this template.
	// DEPRECATED: This field is not used.
	Template string `json:"template,omitempty" protobuf:"bytes,2,opt,name=template"`

	// Arguments hold arguments to the template.
	// DEPRECATED: This field is not used.
	Arguments Arguments `json:"arguments,omitempty" protobuf:"bytes,3,opt,name=arguments"`

	// TemplateRef is the reference to the template resource which is used as the base of this template.
	// DEPRECATED: This field is not used.
	TemplateRef *TemplateRef `json:"templateRef,omitempty" protobuf:"bytes,4,opt,name=templateRef"`

	// Inputs describe what inputs parameters and artifacts are supplied to this template
	Inputs Inputs `json:"inputs,omitempty" protobuf:"bytes,5,opt,name=inputs"`

	// Outputs describe the parameters and artifacts that this template produces
	Outputs Outputs `json:"outputs,omitempty" protobuf:"bytes,6,opt,name=outputs"`

	// NodeSelector is a selector to schedule this step of the workflow to be
	// run on the selected node(s). Overrides the selector set at the workflow level.
	NodeSelector map[string]string `json:"nodeSelector,omitempty" protobuf:"bytes,7,opt,name=nodeSelector"`

	// Affinity sets the pod's scheduling constraints
	// Overrides the affinity set at the workflow level (if any)
	Affinity *apiv1.Affinity `json:"affinity,omitempty" protobuf:"bytes,8,opt,name=affinity"`

	// Metdata sets the pods's metadata, i.e. annotations and labels
	Metadata Metadata `json:"metadata,omitempty" protobuf:"bytes,9,opt,name=metadata"`

	// Deamon will allow a workflow to proceed to the next step so long as the container reaches readiness
	Daemon *bool `json:"daemon,omitempty" protobuf:"bytes,10,opt,name=daemon"`

	// Steps define a series of sequential/parallel workflow steps
	Steps []ParallelSteps `json:"steps,omitempty" protobuf:"bytes,11,opt,name=steps"`

	// Container is the main container image to run in the pod
	Container *apiv1.Container `json:"container,omitempty" protobuf:"bytes,12,opt,name=container"`

	// Script runs a portion of code against an interpreter
	Script *ScriptTemplate `json:"script,omitempty" protobuf:"bytes,13,opt,name=script"`

	// Resource template subtype which can run k8s resources
	Resource *ResourceTemplate `json:"resource,omitempty" protobuf:"bytes,14,opt,name=resource"`

	// DAG template subtype which runs a DAG
	DAG *DAGTemplate `json:"dag,omitempty" protobuf:"bytes,15,opt,name=dag"`

	// Suspend template subtype which can suspend a workflow when reaching the step
	Suspend *SuspendTemplate `json:"suspend,omitempty" protobuf:"bytes,16,opt,name=suspend"`

	// Volumes is a list of volumes that can be mounted by containers in a template.
	// +patchStrategy=merge
	// +patchMergeKey=name
	Volumes []apiv1.Volume `json:"volumes,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,17,opt,name=volumes"`

	// InitContainers is a list of containers which run before the main container.
	// +patchStrategy=merge
	// +patchMergeKey=name
	InitContainers []UserContainer `json:"initContainers,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,18,opt,name=initContainers"`

	// Sidecars is a list of containers which run alongside the main container
	// Sidecars are automatically killed when the main container completes
	// +patchStrategy=merge
	// +patchMergeKey=name
	Sidecars []UserContainer `json:"sidecars,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,19,opt,name=sidecars"`

	// Location in which all files related to the step will be stored (logs, artifacts, etc...).
	// Can be overridden by individual items in Outputs. If omitted, will use the default
	// artifact repository location configured in the controller, appended with the
	// <workflowname>/<nodename> in the key.
	ArchiveLocation *ArtifactLocation `json:"archiveLocation,omitempty" protobuf:"bytes,20,opt,name=archiveLocation"`

	// Optional duration in seconds relative to the StartTime that the pod may be active on a node
	// before the system actively tries to terminate the pod; value must be positive integer
	// This field is only applicable to container and script templates.
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty" protobuf:"bytes,21,opt,name=activeDeadlineSeconds"`

	// RetryStrategy describes how to retry a template when it fails
	RetryStrategy *RetryStrategy `json:"retryStrategy,omitempty" protobuf:"bytes,22,opt,name=retryStrategy"`

	// Parallelism limits the max total parallel pods that can execute at the same time within the
	// boundaries of this template invocation. If additional steps/dag templates are invoked, the
	// pods created by those templates will not be counted towards this total.
	Parallelism *int64 `json:"parallelism,omitempty" protobuf:"bytes,23,opt,name=parallelism"`

	// Tolerations to apply to workflow pods.
	// +patchStrategy=merge
	// +patchMergeKey=key
	Tolerations []apiv1.Toleration `json:"tolerations,omitempty" patchStrategy:"merge" patchMergeKey:"key" protobuf:"bytes,24,opt,name=tolerations"`

	// If specified, the pod will be dispatched by specified scheduler.
	// Or it will be dispatched by workflow scope scheduler if specified.
	// If neither specified, the pod will be dispatched by default scheduler.
	// +optional
	SchedulerName string `json:"schedulerName,omitempty" protobuf:"bytes,25,opt,name=schedulerName"`

	// PriorityClassName to apply to workflow pods.
	PriorityClassName string `json:"priorityClassName,omitempty" protobuf:"bytes,26,opt,name=priorityClassName"`

	// Priority to apply to workflow pods.
	Priority *int32 `json:"priority,omitempty" protobuf:"bytes,27,opt,name=priority"`

	// ServiceAccountName to apply to workflow pods
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,28,opt,name=serviceAccountName"`

	// AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods.
	// ServiceAccountName of ExecutorConfig must be specified if this value is false.
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty" protobuf:"varint,32,opt,name=automountServiceAccountToken"`

	// Executor holds configurations of the executor container.
	Executor *ExecutorConfig `json:"executor,omitempty" protobuf:"bytes,33,opt,name=executor"`

	// HostAliases is an optional list of hosts and IPs that will be injected into the pod spec
	// +patchStrategy=merge
	// +patchMergeKey=ip
	HostAliases []apiv1.HostAlias `json:"hostAliases,omitempty" patchStrategy:"merge" patchMergeKey:"ip" protobuf:"bytes,29,opt,name=hostAliases"`

	// SecurityContext holds pod-level security attributes and common container settings.
	// Optional: Defaults to empty.  See type description for default values of each field.
	// +optional
	SecurityContext *apiv1.PodSecurityContext `json:"securityContext,omitempty" protobuf:"bytes,30,opt,name=securityContext"`

	// PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of
	// container fields which are not strings (e.g. resource limits).
	PodSpecPatch string `json:"podSpecPatch,omitempty" protobuf:"bytes,31,opt,name=podSpecPatch"`

	// ResubmitPendingPods is a flag to enable resubmitting pods that remain Pending after initial submission
	ResubmitPendingPods *bool `json:"resubmitPendingPods,omitempty" protobuf:"varint,34,opt,name=resubmitPendingPods"`

	// Metrics are a list of metrics emitted from this template
	Metrics *Metrics `json:"metrics,omitempty" protobuf:"bytes,35,opt,name=metrics"`
}

// DEPRECATED: Templates should not be used as TemplateReferenceHolder
var _ TemplateReferenceHolder = &Template{}

// DEPRECATED: Templates should not be used as TemplateReferenceHolder
func (tmpl *Template) GetTemplateName() string {
	if tmpl.Template != "" {
		return tmpl.Template
	} else {
		return tmpl.Name
	}
}

// DEPRECATED: Templates should not be used as TemplateReferenceHolder
func (tmpl *Template) GetTemplateRef() *TemplateRef {
	return tmpl.TemplateRef
}

// GetBaseTemplate returns a base template content.
func (tmpl *Template) GetBaseTemplate() *Template {
	baseTemplate := tmpl.DeepCopy()
	baseTemplate.Inputs = Inputs{}
	return baseTemplate
}

func (tmpl *Template) HasPodSpecPatch() bool {
	return tmpl.PodSpecPatch != ""
}

type Artifacts []Artifact

func (a Artifacts) GetArtifactByName(name string) *Artifact {
	for _, art := range a {
		if art.Name == name {
			return &art
		}
	}
	return nil
}

// Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another
type Inputs struct {
	// Parameters are a list of parameters passed as inputs
	// +patchStrategy=merge
	// +patchMergeKey=name
	Parameters []Parameter `json:"parameters,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,1,opt,name=parameters"`

	// Artifact are a list of artifacts passed as inputs
	// +patchStrategy=merge
	// +patchMergeKey=name
	Artifacts Artifacts `json:"artifacts,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,opt,name=artifacts"`
}

func (in Inputs) IsEmpty() bool {
	return len(in.Parameters) == 0 && len(in.Artifacts) == 0
}

// Pod metdata
type Metadata struct {
	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,1,opt,name=annotations"`
	Labels      map[string]string `json:"labels,omitempty" protobuf:"bytes,2,opt,name=labels"`
}

// Parameter indicate a passed string parameter to a service template with an optional default value
type Parameter struct {
	// Name is the parameter name
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Default is the default value to use for an input parameter if a value was not supplied
	// DEPRECATED: This field is not used
	Default *string `json:"default,omitempty" protobuf:"bytes,2,opt,name=default"`

	// Value is the literal value to use for the parameter.
	// If specified in the context of an input parameter, the value takes precedence over any passed values
	Value *string `json:"value,omitempty" protobuf:"bytes,3,opt,name=value"`

	// ValueFrom is the source for the output parameter's value
	ValueFrom *ValueFrom `json:"valueFrom,omitempty" protobuf:"bytes,4,opt,name=valueFrom"`

	// GlobalName exports an output parameter to the global scope, making it available as
	// '{{workflow.outputs.parameters.XXXX}} and in workflow.status.outputs.parameters
	GlobalName string `json:"globalName,omitempty" protobuf:"bytes,5,opt,name=globalName"`
}

func (p *Parameter) UnmarshalJSON(value []byte) error {
	var candidate map[string]interface{}
	err := json.Unmarshal(value, &candidate)
	if err != nil {
		return err
	}
	if val, ok := candidate["name"]; ok {
		p.Name = fmt.Sprint(val)
	}
	if val, ok := candidate["default"]; ok {
		stringVal := fmt.Sprint(val)
		p.Default = &stringVal
	}
	if val, ok := candidate["value"]; ok {
		stringVal := fmt.Sprint(val)
		p.Value = &stringVal
	}
	if val, ok := candidate["valueFrom"]; ok {
		strVal, err := json.Marshal(val)
		if err != nil {
			return err
		}
		err = json.Unmarshal(strVal, &p.ValueFrom)
		if err != nil {
			return err
		}
	}
	if val, ok := candidate["globalName"]; ok {
		p.GlobalName = fmt.Sprint(val)
	}
	return nil
}

// ValueFrom describes a location in which to obtain the value to a parameter
type ValueFrom struct {
	// Path in the container to retrieve an output parameter value from in container templates
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`

	// JSONPath of a resource to retrieve an output parameter value from in resource templates
	JSONPath string `json:"jsonPath,omitempty" protobuf:"bytes,2,opt,name=jsonPath"`

	// JQFilter expression against the resource object in resource templates
	JQFilter string `json:"jqFilter,omitempty" protobuf:"bytes,3,opt,name=jqFilter"`

	// Parameter reference to a step or dag task in which to retrieve an output parameter value from
	// (e.g. '{{steps.mystep.outputs.myparam}}')
	Parameter string `json:"parameter,omitempty" protobuf:"bytes,4,opt,name=parameter"`

	// Default specifies a value to be used if retrieving the value from the specified source fails
	Default *string `json:"default,omitempty" protobuf:"bytes,5,opt,name=default"`
}

// Artifact indicates an artifact to place at a specified path
type Artifact struct {
	// name of the artifact. must be unique within a template's inputs/outputs.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Path is the container path to the artifact
	Path string `json:"path,omitempty" protobuf:"bytes,2,opt,name=path"`

	// mode bits to use on this file, must be a value between 0 and 0777
	// set when loading input artifacts.
	Mode *int32 `json:"mode,omitempty" protobuf:"varint,3,opt,name=mode"`

	// From allows an artifact to reference an artifact from a previous step
	From string `json:"from,omitempty" protobuf:"bytes,4,opt,name=from"`

	// ArtifactLocation contains the location of the artifact
	ArtifactLocation `json:",inline" protobuf:"bytes,5,opt,name=artifactLocation"`

	// GlobalName exports an output artifact to the global scope, making it available as
	// '{{workflow.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts
	GlobalName string `json:"globalName,omitempty" protobuf:"bytes,6,opt,name=globalName"`

	// Archive controls how the artifact will be saved to the artifact repository.
	Archive *ArchiveStrategy `json:"archive,omitempty" protobuf:"bytes,7,opt,name=archive"`

	// Make Artifacts optional, if Artifacts doesn't generate or exist
	Optional bool `json:"optional,omitempty" protobuf:"varint,8,opt,name=optional"`
}

// PodGC describes how to delete completed pods as they complete
type PodGC struct {
	// Strategy is the strategy to use. One of "OnPodCompletion", "OnPodSuccess", "OnWorkflowCompletion", "OnWorkflowSuccess"
	Strategy PodGCStrategy `json:"strategy,omitempty" protobuf:"bytes,1,opt,name=strategy,casttype=PodGCStrategy"`
}

// ArchiveStrategy describes how to archive files/directory when saving artifacts
type ArchiveStrategy struct {
	Tar  *TarStrategy  `json:"tar,omitempty" protobuf:"bytes,1,opt,name=tar"`
	None *NoneStrategy `json:"none,omitempty" protobuf:"bytes,2,opt,name=none"`
}

// TarStrategy will tar and gzip the file or directory when saving
type TarStrategy struct {
	// CompressionLevel specifies the gzip compression level to use for the artifact.
	// Defaults to gzip.DefaultCompression.
	CompressionLevel *int32 `json:"compressionLevel,omitempty" protobuf:"varint,1,opt,name=compressionLevel"`
}

// NoneStrategy indicates to skip tar process and upload the files or directory tree as independent
// files. Note that if the artifact is a directory, the artifact driver must support the ability to
// save/load the directory appropriately.
type NoneStrategy struct{}

// ArtifactLocation describes a location for a single or multiple artifacts.
// It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname).
// It is also used to describe the location of multiple artifacts such as the archive location
// of a single workflow step, which the executor will use as a default location to store its files.
type ArtifactLocation struct {
	// ArchiveLogs indicates if the container logs should be archived
	ArchiveLogs *bool `json:"archiveLogs,omitempty" protobuf:"varint,1,opt,name=archiveLogs"`

	// S3 contains S3 artifact location details
	S3 *S3Artifact `json:"s3,omitempty" protobuf:"bytes,2,opt,name=s3"`

	// Git contains git artifact location details
	Git *GitArtifact `json:"git,omitempty" protobuf:"bytes,3,opt,name=git"`

	// HTTP contains HTTP artifact location details
	HTTP *HTTPArtifact `json:"http,omitempty" protobuf:"bytes,4,opt,name=http"`

	// Artifactory contains artifactory artifact location details
	Artifactory *ArtifactoryArtifact `json:"artifactory,omitempty" protobuf:"bytes,5,opt,name=artifactory"`

	// HDFS contains HDFS artifact location details
	HDFS *HDFSArtifact `json:"hdfs,omitempty" protobuf:"bytes,6,opt,name=hdfs"`

	// Raw contains raw artifact location details
	Raw *RawArtifact `json:"raw,omitempty" protobuf:"bytes,7,opt,name=raw"`

	// OSS contains OSS artifact location details
	OSS *OSSArtifact `json:"oss,omitempty" protobuf:"bytes,8,opt,name=oss"`

	// GCS contains GCS artifact location details
	GCS *GCSArtifact `json:"gcs,omitempty" protobuf:"bytes,9,opt,name=gcs"`
}

// HasLocation whether or not an artifact has a location defined
func (a *ArtifactLocation) HasLocation() bool {
	return a.S3.HasLocation() ||
		a.Git.HasLocation() ||
		a.HTTP.HasLocation() ||
		a.Artifactory.HasLocation() ||
		a.Raw.HasLocation() ||
		a.HDFS.HasLocation() ||
		a.OSS.HasLocation() ||
		a.GCS.HasLocation()
}

type ArtifactRepositoryRef struct {
	ConfigMap string `json:"configMap,omitempty" protobuf:"bytes,1,opt,name=configMap"`
	Key       string `json:"key,omitempty" protobuf:"bytes,2,opt,name=key"`
}

// Outputs hold parameters, artifacts, and results from a step
type Outputs struct {
	// Parameters holds the list of output parameters produced by a step
	// +patchStrategy=merge
	// +patchMergeKey=name
	Parameters []Parameter `json:"parameters,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,1,rep,name=parameters"`

	// Artifacts holds the list of output artifacts produced by a step
	// +patchStrategy=merge
	// +patchMergeKey=name
	Artifacts Artifacts `json:"artifacts,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=artifacts"`

	// Result holds the result (stdout) of a script template
	Result *string `json:"result,omitempty" protobuf:"bytes,3,opt,name=result"`

	// ExitCode holds the exit code of a script template
	ExitCode *string `json:"exitCode,omitempty" protobuf:"bytes,4,opt,name=exitCode"`
}

// WorkflowStep is a reference to a template to execute in a series of step
type WorkflowStep struct {
	// Name of the step
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// Template is the name of the template to execute as the step
	Template string `json:"template,omitempty" protobuf:"bytes,2,opt,name=template"`

	// Arguments hold arguments to the template
	Arguments Arguments `json:"arguments,omitempty" protobuf:"bytes,3,opt,name=arguments"`

	// TemplateRef is the reference to the template resource to execute as the step.
	TemplateRef *TemplateRef `json:"templateRef,omitempty" protobuf:"bytes,4,opt,name=templateRef"`

	// WithItems expands a step into multiple parallel steps from the items in the list
	WithItems []Item `json:"withItems,omitempty" protobuf:"bytes,5,rep,name=withItems"`

	// WithParam expands a step into multiple parallel steps from the value in the parameter,
	// which is expected to be a JSON list.
	WithParam string `json:"withParam,omitempty" protobuf:"bytes,6,opt,name=withParam"`

	// WithSequence expands a step into a numeric sequence
	WithSequence *Sequence `json:"withSequence,omitempty" protobuf:"bytes,7,opt,name=withSequence"`

	// When is an expression in which the step should conditionally execute
	When string `json:"when,omitempty" protobuf:"bytes,8,opt,name=when"`

	// ContinueOn makes argo to proceed with the following step even if this step fails.
	// Errors and Failed states can be specified
	ContinueOn *ContinueOn `json:"continueOn,omitempty" protobuf:"bytes,9,opt,name=continueOn"`

	// OnExit is a template reference which is invoked at the end of the
	// template, irrespective of the success, failure, or error of the
	// primary template.
	OnExit string `json:"onExit,omitempty" protobuf:"bytes,11,opt,name=onExit"`
}

var _ TemplateReferenceHolder = &WorkflowStep{}

func (step *WorkflowStep) GetTemplateName() string {
	return step.Template
}

func (step *WorkflowStep) GetTemplateRef() *TemplateRef {
	return step.TemplateRef
}

func (step *WorkflowStep) ShouldExpand() bool {
	return len(step.WithItems) != 0 || step.WithParam != "" || step.WithSequence != nil
}

// Sequence expands a workflow step into numeric range
type Sequence struct {
	// Count is number of elements in the sequence (default: 0). Not to be used with end
	Count string `json:"count,omitempty" protobuf:"bytes,1,opt,name=count"`

	// Number at which to start the sequence (default: 0)
	Start string `json:"start,omitempty" protobuf:"bytes,2,opt,name=start"`

	// Number at which to end the sequence (default: 0). Not to be used with Count
	End string `json:"end,omitempty" protobuf:"bytes,3,opt,name=end"`

	// Format is a printf format string to format the value in the sequence
	Format string `json:"format,omitempty" protobuf:"bytes,4,opt,name=format"`
}

// DeepCopyInto is an custom deepcopy function to deal with our use of the interface{} type
func (i *Item) DeepCopyInto(out *Item) {
	inBytes, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(inBytes, out)
	if err != nil {
		panic(err)
	}
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (i Item) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
func (i Item) OpenAPISchemaFormat() string { return "item" }

// TemplateRef is a reference of template resource.
type TemplateRef struct {
	// Name is the resource name of the template.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Template is the name of referred template in the resource.
	Template string `json:"template,omitempty" protobuf:"bytes,2,opt,name=template"`
	// RuntimeResolution skips validation at creation time.
	// By enabling this option, you can create the referred workflow template before the actual runtime.
	RuntimeResolution bool `json:"runtimeResolution,omitempty" protobuf:"varint,3,opt,name=runtimeResolution"`
	// ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate).
	ClusterScope bool `json:"clusterScope,omitempty" protobuf:"varint,4,opt,name=clusterScope"`
}

// WorkflowTemplateRef is a reference to a WorkflowTemplate resource.
type WorkflowTemplateRef struct {
	// Name is the resource name of the workflow template.
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate).
	ClusterScope bool `json:"clusterScope,omitempty" protobuf:"varint,2,opt,name=clusterScope"`
}

func (ref *WorkflowTemplateRef) ToTemplateRef(entrypoint string) *TemplateRef {
	return &TemplateRef{
		Name:         ref.Name,
		ClusterScope: ref.ClusterScope,
		Template:     entrypoint,
	}
}

type ArgumentsProvider interface {
	GetParameterByName(name string) *Parameter
	GetArtifactByName(name string) *Artifact
}

// Arguments to a template
type Arguments struct {
	// Parameters is the list of parameters to pass to the template or workflow
	// +patchStrategy=merge
	// +patchMergeKey=name
	Parameters []Parameter `json:"parameters,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,1,rep,name=parameters"`

	// Artifacts is the list of artifacts to pass to the template or workflow
	// +patchStrategy=merge
	// +patchMergeKey=name
	Artifacts Artifacts `json:"artifacts,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=artifacts"`
}

func (a Arguments) IsEmpty() bool {
	return len(a.Parameters) == 0 && len(a.Artifacts) == 0
}

var _ ArgumentsProvider = &Arguments{}

type Nodes map[string]NodeStatus

func (n Nodes) FindByDisplayName(name string) *NodeStatus {
	for _, i := range n {
		if i.DisplayName == name {
			return &i
		}
	}
	return nil
}

func (in Nodes) Any(f func(node NodeStatus) bool) bool {
	for _, i := range in {
		if f(i) {
			return true
		}
	}
	return false
}

// UserContainer is a container specified by a user.
type UserContainer struct {
	apiv1.Container `json:",inline" protobuf:"bytes,1,opt,name=container"`

	// MirrorVolumeMounts will mount the same volumes specified in the main container
	// to the container (including artifacts), at the same mountPaths. This enables
	// dind daemon to partially see the same filesystem as the main container in
	// order to use features such as docker volume binding
	MirrorVolumeMounts *bool `json:"mirrorVolumeMounts,omitempty" protobuf:"varint,2,opt,name=mirrorVolumeMounts"`
}

// WorkflowStatus contains overall status information about a workflow
type WorkflowStatus struct {
	// Phase a simple, high-level summary of where the workflow is in its lifecycle.
	Phase NodePhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase,casttype=NodePhase"`

	// Time at which this workflow started
	StartedAt metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,2,opt,name=startedAt"`

	// Time at which this workflow completed
	FinishedAt metav1.Time `json:"finishedAt,omitempty" protobuf:"bytes,3,opt,name=finishedAt"`

	// A human readable message indicating details about why the workflow is in this condition.
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`

	// Compressed and base64 decoded Nodes map
	CompressedNodes string `json:"compressedNodes,omitempty" protobuf:"bytes,5,opt,name=compressedNodes"`

	// Nodes is a mapping between a node ID and the node's status.
	Nodes Nodes `json:"nodes,omitempty" protobuf:"bytes,6,rep,name=nodes"`

	// Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty.
	// This will actually be populated with a hash of the offloaded data.
	OffloadNodeStatusVersion string `json:"offloadNodeStatusVersion,omitempty" protobuf:"bytes,10,rep,name=offloadNodeStatusVersion"`

	// StoredTemplates is a mapping between a template ref and the node's status.
	StoredTemplates map[string]Template `json:"storedTemplates,omitempty" protobuf:"bytes,9,rep,name=storedTemplates"`

	// PersistentVolumeClaims tracks all PVCs that were created as part of the workflow.
	// The contents of this list are drained at the end of the workflow.
	PersistentVolumeClaims []apiv1.Volume `json:"persistentVolumeClaims,omitempty" protobuf:"bytes,7,rep,name=persistentVolumeClaims"`

	// Outputs captures output values and artifact locations produced by the workflow via global outputs
	Outputs *Outputs `json:"outputs,omitempty" protobuf:"bytes,8,opt,name=outputs"`

	// Conditions is a list of conditions the Workflow may have
	Conditions Conditions `json:"conditions,omitempty" protobuf:"bytes,13,rep,name=conditions"`

	// ResourcesDuration is the total for the workflow
	ResourcesDuration ResourcesDuration `json:"resourcesDuration,omitempty" protobuf:"bytes,12,opt,name=resourcesDuration"`

	// StoredWorkflowSpec stores the WorkflowTemplate spec for future execution.
	StoredWorkflowSpec *WorkflowSpec `json:"storedWorkflowTemplateSpec,omitempty" protobuf:"bytes,14,opt,name=storedWorkflowTemplateSpec"`
}

func (ws *WorkflowStatus) IsOffloadNodeStatus() bool {
	return ws.OffloadNodeStatusVersion != ""
}

func (ws *WorkflowStatus) GetOffloadNodeStatusVersion() string {
	return ws.OffloadNodeStatusVersion
}

func (wf *Workflow) GetOffloadNodeStatusVersion() string {
	return wf.Status.GetOffloadNodeStatusVersion()
}

type RetryPolicy string

const (
	RetryPolicyAlways    RetryPolicy = "Always"
	RetryPolicyOnFailure RetryPolicy = "OnFailure"
	RetryPolicyOnError   RetryPolicy = "OnError"
)

// Backoff is a backoff strategy to use within retryStrategy
type Backoff struct {
	// Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. "2m", "1h")
	Duration string `json:"duration,omitempty" protobuf:"varint,1,opt,name=duration"`
	// Factor is a factor to multiply the base duration after each failed retry
	Factor int32 `json:"factor,omitempty" protobuf:"varint,2,opt,name=factor"`
	// MaxDuration is the maximum amount of time allowed for the backoff strategy
	MaxDuration string `json:"maxDuration,omitempty" protobuf:"varint,3,opt,name=maxDuration"`
}

// RetryStrategy provides controls on how to retry a workflow step
type RetryStrategy struct {
	// Limit is the maximum number of attempts when retrying a container
	Limit *int32 `json:"limit,omitempty" protobuf:"varint,1,opt,name=limit"`

	// RetryPolicy is a policy of NodePhase statuses that will be retried
	RetryPolicy RetryPolicy `json:"retryPolicy,omitempty" protobuf:"bytes,2,opt,name=retryPolicy,casttype=RetryPolicy"`

	// Backoff is a backoff strategy
	Backoff *Backoff `json:"backoff,omitempty" protobuf:"bytes,3,opt,name=backoff,casttype=Backoff"`
}

// The amount of requested resource * the duration that request was used.
// This is represented as duration in seconds, so can be converted to and from
// duration (with loss of precision).
type ResourceDuration int64

func NewResourceDuration(d time.Duration) ResourceDuration {
	return ResourceDuration(d.Seconds())
}

func (in ResourceDuration) Duration() time.Duration {
	return time.Duration(in) * time.Second
}

func (in ResourceDuration) String() string {
	return in.Duration().String()
}

// This contains each duration by request requested.
// e.g. 100m CPU * 1h, 1Gi memory * 1h
type ResourcesDuration map[apiv1.ResourceName]ResourceDuration

func (in ResourcesDuration) Add(o ResourcesDuration) ResourcesDuration {
	for n, d := range o {
		in[n] += d
	}
	return in
}

func (in ResourcesDuration) String() string {
	var parts []string
	for n, d := range in {
		parts = append(parts, fmt.Sprintf("%v*(%s %s)", d, ResourceQuantityDenominator(n).String(), n))
	}
	return strings.Join(parts, ",")
}

func (in ResourcesDuration) IsZero() bool {
	return len(in) == 0
}

func ResourceQuantityDenominator(r apiv1.ResourceName) *resource.Quantity {
	q, ok := map[apiv1.ResourceName]resource.Quantity{
		apiv1.ResourceMemory:           resource.MustParse("100Mi"),
		apiv1.ResourceStorage:          resource.MustParse("10Gi"),
		apiv1.ResourceEphemeralStorage: resource.MustParse("10Gi"),
	}[r]
	if !ok {
		q = resource.MustParse("1")
	}
	return &q
}

type Conditions []Condition

func (cs *Conditions) UpsertCondition(condition Condition) {
	for index, wfCondition := range *cs {
		if wfCondition.Type == condition.Type {
			(*cs)[index] = condition
			return
		}
	}
	*cs = append(*cs, condition)
}

func (cs *Conditions) UpsertConditionMessage(condition Condition) {
	for index, wfCondition := range *cs {
		if wfCondition.Type == condition.Type {
			(*cs)[index].Message += ", " + condition.Message
			return
		}
	}
	*cs = append(*cs, condition)
}

func (cs *Conditions) JoinConditions(conditions *Conditions) {
	for _, condition := range *conditions {
		cs.UpsertCondition(condition)
	}
}

func (cs *Conditions) RemoveCondition(conditionType ConditionType) {
	for index, wfCondition := range *cs {
		if wfCondition.Type == conditionType {
			*cs = append((*cs)[:index], (*cs)[index+1:]...)
			return
		}
	}
}

func (cs *Conditions) DisplayString(fmtStr string, iconMap map[ConditionType]string) string {
	if len(*cs) == 0 {
		return fmt.Sprintf(fmtStr, "Conditions:", "None")
	}
	out := fmt.Sprintf(fmtStr, "Conditions:", "")
	for _, condition := range *cs {
		conditionMessage := condition.Message
		if conditionMessage == "" {
			conditionMessage = string(condition.Status)
		}
		conditionPrefix := fmt.Sprintf("%s %s", iconMap[condition.Type], string(condition.Type))
		out += fmt.Sprintf(fmtStr, conditionPrefix, conditionMessage)
	}
	return out
}

type ConditionType string

const (
	// ConditionTypeCompleted is a signifies the workflow has completed
	ConditionTypeCompleted ConditionType = "Completed"
	// ConditionTypeSpecWarning is a warning on the current application spec
	ConditionTypeSpecWarning ConditionType = "SpecWarning"
	// ConditionTypeMetricsError is an error during metric emission
	ConditionTypeMetricsError ConditionType = "MetricsError"
)

type Condition struct {
	// Type is the type of condition
	Type ConditionType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type,casttype=ConditionType"`

	// Status is the status of the condition
	Status metav1.ConditionStatus `json:"status,omitempty" protobuf:"bytes,2,opt,name=status,casttype=k8s.io/apimachinery/pkg/apis/meta/v1.ConditionStatus"`

	// Message is the condition message
	Message string `json:"message,omitempty" protobuf:"bytes,3,opt,name=message"`
}

// NodeStatus contains status information about an individual node in the workflow
type NodeStatus struct {
	// ID is a unique identifier of a node within the worklow
	// It is implemented as a hash of the node name, which makes the ID deterministic
	ID string `json:"id" protobuf:"bytes,1,opt,name=id"`

	// Name is unique name in the node tree used to generate the node ID
	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`

	// DisplayName is a human readable representation of the node. Unique within a template boundary
	DisplayName string `json:"displayName" protobuf:"bytes,3,opt,name=displayName"`

	// Type indicates type of node
	Type NodeType `json:"type" protobuf:"bytes,4,opt,name=type,casttype=NodeType"`

	// TemplateName is the template name which this node corresponds to.
	// Not applicable to virtual nodes (e.g. Retry, StepGroup)
	TemplateName string `json:"templateName,omitempty" protobuf:"bytes,5,opt,name=templateName"`

	// TemplateRef is the reference to the template resource which this node corresponds to.
	// Not applicable to virtual nodes (e.g. Retry, StepGroup)
	TemplateRef *TemplateRef `json:"templateRef,omitempty" protobuf:"bytes,6,opt,name=templateRef"`

	// StoredTemplateID is the ID of stored template.
	// DEPRECATED: This value is not used anymore.
	StoredTemplateID string `json:"storedTemplateID,omitempty" protobuf:"bytes,18,opt,name=storedTemplateID"`

	// WorkflowTemplateName is the WorkflowTemplate resource name on which the resolved template of this node is retrieved.
	// DEPRECATED: This value is not used anymore.
	WorkflowTemplateName string `json:"workflowTemplateName,omitempty" protobuf:"bytes,19,opt,name=workflowTemplateName"`

	// TemplateScope is the template scope in which the template of this node was retrieved.
	TemplateScope string `json:"templateScope,omitempty" protobuf:"bytes,20,opt,name=templateScope"`

	// Phase a simple, high-level summary of where the node is in its lifecycle.
	// Can be used as a state machine.
	Phase NodePhase `json:"phase,omitempty" protobuf:"bytes,7,opt,name=phase,casttype=NodePhase"`

	// BoundaryID indicates the node ID of the associated template root node in which this node belongs to
	BoundaryID string `json:"boundaryID,omitempty" protobuf:"bytes,8,opt,name=boundaryID"`

	// A human readable message indicating details about why the node is in this condition.
	Message string `json:"message,omitempty" protobuf:"bytes,9,opt,name=message"`

	// Time at which this node started
	StartedAt metav1.Time `json:"startedAt,omitempty" protobuf:"bytes,10,opt,name=startedAt"`

	// Time at which this node completed
	FinishedAt metav1.Time `json:"finishedAt,omitempty" protobuf:"bytes,11,opt,name=finishedAt"`

	// ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes.
	ResourcesDuration ResourcesDuration `json:"resourcesDuration,omitempty" protobuf:"bytes,21,opt,name=resourcesDuration"`

	// PodIP captures the IP of the pod for daemoned steps
	PodIP string `json:"podIP,omitempty" protobuf:"bytes,12,opt,name=podIP"`

	// Daemoned tracks whether or not this node was daemoned and need to be terminated
	Daemoned *bool `json:"daemoned,omitempty" protobuf:"varint,13,opt,name=daemoned"`

	// Inputs captures input parameter values and artifact locations supplied to this template invocation
	Inputs *Inputs `json:"inputs,omitempty" protobuf:"bytes,14,opt,name=inputs"`

	// Outputs captures output parameter values and artifact locations produced by this template invocation
	Outputs *Outputs `json:"outputs,omitempty" protobuf:"bytes,15,opt,name=outputs"`

	// Children is a list of child node IDs
	Children []string `json:"children,omitempty" protobuf:"bytes,16,rep,name=children"`

	// OutboundNodes tracks the node IDs which are considered "outbound" nodes to a template invocation.
	// For every invocation of a template, there are nodes which we considered as "outbound". Essentially,
	// these are last nodes in the execution sequence to run, before the template is considered completed.
	// These nodes are then connected as parents to a following step.
	//
	// In the case of single pod steps (i.e. container, script, resource templates), this list will be nil
	// since the pod itself is already considered the "outbound" node.
	// In the case of DAGs, outbound nodes are the "target" tasks (tasks with no children).
	// In the case of steps, outbound nodes are all the containers involved in the last step group.
	// NOTE: since templates are composable, the list of outbound nodes are carried upwards when
	// a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of
	// a template, will be a superset of the outbound nodes of its last children.
	OutboundNodes []string `json:"outboundNodes,omitempty" protobuf:"bytes,17,rep,name=outboundNodes"`

	// HostNodeName name of the Kubernetes node on which the Pod is running, if applicable
	HostNodeName string `json:"hostNodeName,omitempty" protobuf:"bytes,22,rep,name=hostNodeName"`
}

func (n Nodes) GetResourcesDuration() ResourcesDuration {
	i := ResourcesDuration{}
	for _, status := range n {
		i = i.Add(status.ResourcesDuration)
	}
	return i
}

func (phase NodePhase) Completed() bool {
	return phase == NodeSucceeded ||
		phase == NodeFailed ||
		phase == NodeError ||
		phase == NodeSkipped
}

// Completed returns whether or not the workflow has completed execution
func (ws WorkflowStatus) Completed() bool {
	return ws.Phase.Completed()
}

// Successful return whether or not the workflow has succeeded
func (ws WorkflowStatus) Successful() bool {
	return ws.Phase == NodeSucceeded
}

// Failed return whether or not the workflow has failed
func (ws WorkflowStatus) Failed() bool {
	return ws.Phase == NodeFailed
}

func (ws WorkflowStatus) StartTime() *metav1.Time {
	return &ws.StartedAt
}

func (ws WorkflowStatus) FinishTime() *metav1.Time {
	return &ws.FinishedAt
}

// Completed returns whether or not the node has completed execution
func (n NodeStatus) Completed() bool {
	return n.Phase.Completed() || n.IsDaemoned() && n.Phase != NodePending
}

func (in *WorkflowStatus) AnyActiveSuspendNode() bool {
	return in.Nodes.Any(func(node NodeStatus) bool { return node.IsActiveSuspendNode() })
}

// Pending returns whether or not the node is in pending state
func (n NodeStatus) Pending() bool {
	return n.Phase == NodePending
}

// IsDaemoned returns whether or not the node is deamoned
func (n NodeStatus) IsDaemoned() bool {
	if n.Daemoned == nil || !*n.Daemoned {
		return false
	}
	return true
}

// Successful returns whether or not this node completed successfully
func (n NodeStatus) Successful() bool {
	return n.Phase == NodeSucceeded || n.Phase == NodeSkipped || n.IsDaemoned() && n.Phase != NodePending
}

func (n NodeStatus) Failed() bool {
	return !n.Successful()
}

func (n NodeStatus) StartTime() *metav1.Time {
	return &n.StartedAt
}

func (n NodeStatus) FinishTime() *metav1.Time {
	return &n.FinishedAt
}

// CanRetry returns whether the node should be retried or not.
func (n NodeStatus) CanRetry() bool {
	// TODO(shri): Check if there are some 'unretryable' errors.
	return n.Completed() && !n.Successful()
}

func (n NodeStatus) GetTemplateScope() (ResourceScope, string) {
	// For compatibility: an empty TemplateScope is a local scope
	if n.TemplateScope == "" {
		return ResourceScopeLocal, ""
	}
	split := strings.Split(n.TemplateScope, "/")
	// For compatibility: an unspecified ResourceScope in a TemplateScope is a namespaced scope
	if len(split) == 1 {
		return ResourceScopeNamespaced, split[0]
	}
	resourceScope, resourceName := split[0], split[1]
	return ResourceScope(resourceScope), resourceName
}

var _ TemplateReferenceHolder = &NodeStatus{}

func (n *NodeStatus) GetTemplateName() string {
	return n.TemplateName
}

func (n *NodeStatus) GetTemplateRef() *TemplateRef {
	return n.TemplateRef
}

// IsActiveSuspendNode returns whether this node is an active suspend node
func (n *NodeStatus) IsActiveSuspendNode() bool {
	return n.Type == NodeTypeSuspend && n.Phase == NodeRunning
}

// S3Bucket contains the access information required for interfacing with an S3 bucket
type S3Bucket struct {
	// Endpoint is the hostname of the bucket endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`

	// Bucket is the name of the bucket
	Bucket string `json:"bucket" protobuf:"bytes,2,opt,name=bucket"`

	// Region contains the optional bucket region
	Region string `json:"region,omitempty" protobuf:"bytes,3,opt,name=region"`

	// Insecure will connect to the service with TLS
	Insecure *bool `json:"insecure,omitempty" protobuf:"varint,4,opt,name=insecure"`

	// AccessKeySecret is the secret selector to the bucket's access key
	AccessKeySecret apiv1.SecretKeySelector `json:"accessKeySecret" protobuf:"bytes,5,opt,name=accessKeySecret"`

	// SecretKeySecret is the secret selector to the bucket's secret key
	SecretKeySecret apiv1.SecretKeySelector `json:"secretKeySecret" protobuf:"bytes,6,opt,name=secretKeySecret"`

	// RoleARN is the Amazon Resource Name (ARN) of the role to assume.
	RoleARN string `json:"roleARN,omitempty" protobuf:"bytes,7,opt,name=roleARN"`

	// UseSDKCreds tells the driver to figure out credentials based on sdk defaults.
	UseSDKCreds bool `json:"useSDKCreds,omitempty" protobuf:"varint,8,opt,name=useSDKCreds"`
}

// S3Artifact is the location of an S3 artifact
type S3Artifact struct {
	S3Bucket `json:",inline" protobuf:"bytes,1,opt,name=s3Bucket"`

	// Key is the key in the bucket where the artifact resides
	Key string `json:"key" protobuf:"bytes,2,opt,name=key"`
}

func (s *S3Artifact) HasLocation() bool {
	return s != nil && s.Endpoint != "" && s.Bucket != "" && s.Key != ""
}

// GitArtifact is the location of an git artifact
type GitArtifact struct {
	// Repo is the git repository
	Repo string `json:"repo" protobuf:"bytes,1,opt,name=repo"`

	// Revision is the git commit, tag, branch to checkout
	Revision string `json:"revision,omitempty" protobuf:"bytes,2,opt,name=revision"`

	// Depth specifies clones/fetches should be shallow and include the given
	// number of commits from the branch tip
	Depth *uint64 `json:"depth,omitempty" protobuf:"bytes,3,opt,name=depth"`

	// Fetch specifies a number of refs that should be fetched before checkout
	Fetch []string `json:"fetch,omitempty" protobuf:"bytes,4,rep,name=fetch"`

	// UsernameSecret is the secret selector to the repository username
	UsernameSecret *apiv1.SecretKeySelector `json:"usernameSecret,omitempty" protobuf:"bytes,5,opt,name=usernameSecret"`

	// PasswordSecret is the secret selector to the repository password
	PasswordSecret *apiv1.SecretKeySelector `json:"passwordSecret,omitempty" protobuf:"bytes,6,opt,name=passwordSecret"`

	// SSHPrivateKeySecret is the secret selector to the repository ssh private key
	SSHPrivateKeySecret *apiv1.SecretKeySelector `json:"sshPrivateKeySecret,omitempty" protobuf:"bytes,7,opt,name=sshPrivateKeySecret"`

	// InsecureIgnoreHostKey disables SSH strict host key checking during git clone
	InsecureIgnoreHostKey bool `json:"insecureIgnoreHostKey,omitempty" protobuf:"varint,8,opt,name=insecureIgnoreHostKey"`
}

func (g *GitArtifact) HasLocation() bool {
	return g != nil && g.Repo != ""
}

// ArtifactoryAuth describes the secret selectors required for authenticating to artifactory
type ArtifactoryAuth struct {
	// UsernameSecret is the secret selector to the repository username
	UsernameSecret *apiv1.SecretKeySelector `json:"usernameSecret,omitempty" protobuf:"bytes,1,opt,name=usernameSecret"`

	// PasswordSecret is the secret selector to the repository password
	PasswordSecret *apiv1.SecretKeySelector `json:"passwordSecret,omitempty" protobuf:"bytes,2,opt,name=passwordSecret"`
}

// ArtifactoryArtifact is the location of an artifactory artifact
type ArtifactoryArtifact struct {
	// URL of the artifact
	URL             string `json:"url" protobuf:"bytes,1,opt,name=url"`
	ArtifactoryAuth `json:",inline" protobuf:"bytes,2,opt,name=artifactoryAuth"`
}

//func (a *ArtifactoryArtifact) String() string {
//	return a.URL
//}

func (a *ArtifactoryArtifact) HasLocation() bool {
	return a != nil && a.URL != ""
}

// HDFSArtifact is the location of an HDFS artifact
type HDFSArtifact struct {
	HDFSConfig `json:",inline" protobuf:"bytes,1,opt,name=hDFSConfig"`

	// Path is a file path in HDFS
	Path string `json:"path" protobuf:"bytes,2,opt,name=path"`

	// Force copies a file forcibly even if it exists (default: false)
	Force bool `json:"force,omitempty" protobuf:"varint,3,opt,name=force"`
}

func (h *HDFSArtifact) HasLocation() bool {
	return h != nil && len(h.Addresses) > 0
}

// HDFSConfig is configurations for HDFS
type HDFSConfig struct {
	HDFSKrbConfig `json:",inline" protobuf:"bytes,1,opt,name=hDFSKrbConfig"`

	// Addresses is accessible addresses of HDFS name nodes
	Addresses []string `json:"addresses" protobuf:"bytes,2,rep,name=addresses"`

	// HDFSUser is the user to access HDFS file system.
	// It is ignored if either ccache or keytab is used.
	HDFSUser string `json:"hdfsUser,omitempty" protobuf:"bytes,3,opt,name=hdfsUser"`
}

// HDFSKrbConfig is auth configurations for Kerberos
type HDFSKrbConfig struct {
	// KrbCCacheSecret is the secret selector for Kerberos ccache
	// Either ccache or keytab can be set to use Kerberos.
	KrbCCacheSecret *apiv1.SecretKeySelector `json:"krbCCacheSecret,omitempty" protobuf:"bytes,1,opt,name=krbCCacheSecret"`

	// KrbKeytabSecret is the secret selector for Kerberos keytab
	// Either ccache or keytab can be set to use Kerberos.
	KrbKeytabSecret *apiv1.SecretKeySelector `json:"krbKeytabSecret,omitempty" protobuf:"bytes,2,opt,name=krbKeytabSecret"`

	// KrbUsername is the Kerberos username used with Kerberos keytab
	// It must be set if keytab is used.
	KrbUsername string `json:"krbUsername,omitempty" protobuf:"bytes,3,opt,name=krbUsername"`

	// KrbRealm is the Kerberos realm used with Kerberos keytab
	// It must be set if keytab is used.
	KrbRealm string `json:"krbRealm,omitempty" protobuf:"bytes,4,opt,name=krbRealm"`

	// KrbConfig is the configmap selector for Kerberos config as string
	// It must be set if either ccache or keytab is used.
	KrbConfigConfigMap *apiv1.ConfigMapKeySelector `json:"krbConfigConfigMap,omitempty" protobuf:"bytes,5,opt,name=krbConfigConfigMap"`

	// KrbServicePrincipalName is the principal name of Kerberos service
	// It must be set if either ccache or keytab is used.
	KrbServicePrincipalName string `json:"krbServicePrincipalName,omitempty" protobuf:"bytes,6,opt,name=krbServicePrincipalName"`
}

// RawArtifact allows raw string content to be placed as an artifact in a container
type RawArtifact struct {
	// Data is the string contents of the artifact
	Data string `json:"data" protobuf:"bytes,1,opt,name=data"`
}

func (r *RawArtifact) HasLocation() bool {
	return r != nil
}

// HTTPArtifact allows an file served on HTTP to be placed as an input artifact in a container
type HTTPArtifact struct {
	// URL of the artifact
	URL string `json:"url" protobuf:"bytes,1,opt,name=url"`
}

func (h *HTTPArtifact) HasLocation() bool {
	return h != nil && h.URL != ""
}

// GCSBucket contains the access information for interfacring with a GCS bucket
type GCSBucket struct {

	// Bucket is the name of the bucket
	Bucket string `json:"bucket" protobuf:"bytes,1,opt,name=bucket"`

	// ServiceAccountKeySecret is the secret selector to the bucket's service account key
	ServiceAccountKeySecret apiv1.SecretKeySelector `json:"serviceAccountKeySecret,omitempty" protobuf:"bytes,2,opt,name=serviceAccountKeySecret"`
}

// GCSArtifact is the location of a GCS artifact
type GCSArtifact struct {
	GCSBucket `json:",inline" protobuf:"bytes,1,opt,name=gCSBucket"`

	// Key is the path in the bucket where the artifact resides
	Key string `json:"key" protobuf:"bytes,2,opt,name=key"`
}

func (g *GCSArtifact) HasLocation() bool {
	return g != nil && g.Bucket != "" && g.Key != ""
}

// OSSBucket contains the access information required for interfacing with an OSS bucket
type OSSBucket struct {
	// Endpoint is the hostname of the bucket endpoint
	Endpoint string `json:"endpoint" protobuf:"bytes,1,opt,name=endpoint"`

	// Bucket is the name of the bucket
	Bucket string `json:"bucket" protobuf:"bytes,2,opt,name=bucket"`

	// AccessKeySecret is the secret selector to the bucket's access key
	AccessKeySecret apiv1.SecretKeySelector `json:"accessKeySecret" protobuf:"bytes,3,opt,name=accessKeySecret"`

	// SecretKeySecret is the secret selector to the bucket's secret key
	SecretKeySecret apiv1.SecretKeySelector `json:"secretKeySecret" protobuf:"bytes,4,opt,name=secretKeySecret"`
}

// OSSArtifact is the location of an OSS artifact
type OSSArtifact struct {
	OSSBucket `json:",inline" protobuf:"bytes,1,opt,name=oSSBucket"`

	// Key is the path in the bucket where the artifact resides
	Key string `json:"key" protobuf:"bytes,2,opt,name=key"`
}

func (o *OSSArtifact) HasLocation() bool {
	return o != nil && o.Bucket != "" && o.Endpoint != "" && o.Key != ""
}

// ExecutorConfig holds configurations of an executor container.
type ExecutorConfig struct {
	// ServiceAccountName specifies the service account name of the executor container.
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,1,opt,name=serviceAccountName"`
}

// ScriptTemplate is a template subtype to enable scripting through code steps
type ScriptTemplate struct {
	apiv1.Container `json:",inline" protobuf:"bytes,1,opt,name=container"`

	// Source contains the source code of the script to execute
	Source string `json:"source" protobuf:"bytes,2,opt,name=source"`
}

// ResourceTemplate is a template subtype to manipulate kubernetes resources
type ResourceTemplate struct {
	// Action is the action to perform to the resource.
	// Must be one of: get, create, apply, delete, replace, patch
	Action string `json:"action" protobuf:"bytes,1,opt,name=action"`

	// MergeStrategy is the strategy used to merge a patch. It defaults to "strategic"
	// Must be one of: strategic, merge, json
	MergeStrategy string `json:"mergeStrategy,omitempty" protobuf:"bytes,2,opt,name=mergeStrategy"`

	// Manifest contains the kubernetes manifest
	Manifest string `json:"manifest" protobuf:"bytes,3,opt,name=manifest"`

	// SetOwnerReference sets the reference to the workflow on the OwnerReference of generated resource.
	SetOwnerReference bool `json:"setOwnerReference,omitempty" protobuf:"varint,4,opt,name=setOwnerReference"`

	// SuccessCondition is a label selector expression which describes the conditions
	// of the k8s resource in which it is acceptable to proceed to the following step
	SuccessCondition string `json:"successCondition,omitempty" protobuf:"bytes,5,opt,name=successCondition"`

	// FailureCondition is a label selector expression which describes the conditions
	// of the k8s resource in which the step was considered failed
	FailureCondition string `json:"failureCondition,omitempty" protobuf:"bytes,6,opt,name=failureCondition"`

	// Flags is a set of additional options passed to kubectl before submitting a resource
	// I.e. to disable resource validation:
	// flags: [
	// 	"--validate=false"  # disable resource validation
	// ]
	Flags []string `json:"flags,omitempty" protobuf:"varint,7,opt,name=flags"`
}

// GetType returns the type of this template
func (tmpl *Template) GetType() TemplateType {
	if tmpl.Container != nil {
		return TemplateTypeContainer
	}
	if tmpl.Steps != nil {
		return TemplateTypeSteps
	}
	if tmpl.DAG != nil {
		return TemplateTypeDAG
	}
	if tmpl.Script != nil {
		return TemplateTypeScript
	}
	if tmpl.Resource != nil {
		return TemplateTypeResource
	}
	if tmpl.Suspend != nil {
		return TemplateTypeSuspend
	}
	return TemplateTypeUnknown
}

// IsPodType returns whether or not the template is a pod type
func (tmpl *Template) IsPodType() bool {
	switch tmpl.GetType() {
	case TemplateTypeContainer, TemplateTypeScript, TemplateTypeResource:
		return true
	}
	return false
}

// IsLeaf returns whether or not the template is a leaf
func (tmpl *Template) IsLeaf() bool {
	switch tmpl.GetType() {
	case TemplateTypeContainer, TemplateTypeScript, TemplateTypeResource:
		return true
	}
	return false
}

// DAGTemplate is a template subtype for directed acyclic graph templates
type DAGTemplate struct {
	// Target are one or more names of targets to execute in a DAG
	Target string `json:"target,omitempty" protobuf:"bytes,1,opt,name=target"`

	// Tasks are a list of DAG tasks
	// +patchStrategy=merge
	// +patchMergeKey=name
	Tasks []DAGTask `json:"tasks" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=tasks"`

	// This flag is for DAG logic. The DAG logic has a built-in "fail fast" feature to stop scheduling new steps,
	// as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completed
	// before failing the DAG itself.
	// The FailFast flag default is true,  if set to false, it will allow a DAG to run all branches of the DAG to
	// completion (either success or failure), regardless of the failed outcomes of branches in the DAG.
	// More info and example about this feature at https://github.com/argoproj/argo/issues/1442
	FailFast *bool `json:"failFast,omitempty" protobuf:"varint,3,opt,name=failFast"`
}

// DAGTask represents a node in the graph during DAG execution
type DAGTask struct {
	// Name is the name of the target
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// Name of template to execute
	Template string `json:"template" protobuf:"bytes,2,opt,name=template"`

	// Arguments are the parameter and artifact arguments to the template
	Arguments Arguments `json:"arguments,omitempty" protobuf:"bytes,3,opt,name=arguments"`

	// TemplateRef is the reference to the template resource to execute.
	TemplateRef *TemplateRef `json:"templateRef,omitempty" protobuf:"bytes,4,opt,name=templateRef"`

	// Dependencies are name of other targets which this depends on
	Dependencies []string `json:"dependencies,omitempty" protobuf:"bytes,5,rep,name=dependencies"`

	// WithItems expands a task into multiple parallel tasks from the items in the list
	WithItems []Item `json:"withItems,omitempty" protobuf:"bytes,6,rep,name=withItems"`

	// WithParam expands a task into multiple parallel tasks from the value in the parameter,
	// which is expected to be a JSON list.
	WithParam string `json:"withParam,omitempty" protobuf:"bytes,7,opt,name=withParam"`

	// WithSequence expands a task into a numeric sequence
	WithSequence *Sequence `json:"withSequence,omitempty" protobuf:"bytes,8,opt,name=withSequence"`

	// When is an expression in which the task should conditionally execute
	When string `json:"when,omitempty" protobuf:"bytes,9,opt,name=when"`

	// ContinueOn makes argo to proceed with the following step even if this step fails.
	// Errors and Failed states can be specified
	ContinueOn *ContinueOn `json:"continueOn,omitempty" protobuf:"bytes,10,opt,name=continueOn"`

	// OnExit is a template reference which is invoked at the end of the
	// template, irrespective of the success, failure, or error of the
	// primary template.
	OnExit string `json:"onExit,omitempty" protobuf:"bytes,11,opt,name=onExit"`

	// Depends are name of other targets which this depends on
	Depends string `json:"depends,omitempty" protobuf:"bytes,12,opt,name=depends"`
}

var _ TemplateReferenceHolder = &DAGTask{}

func (t *DAGTask) GetTemplateName() string {
	return t.Template
}

func (t *DAGTask) GetTemplateRef() *TemplateRef {
	return t.TemplateRef
}

func (t *DAGTask) ShouldExpand() bool {
	return len(t.WithItems) != 0 || t.WithParam != "" || t.WithSequence != nil
}

// SuspendTemplate is a template subtype to suspend a workflow at a predetermined point in time
type SuspendTemplate struct {
	// Duration is the seconds to wait before automatically resuming a template
	Duration string `json:"duration,omitempty" protobuf:"bytes,1,opt,name=duration"`
}

// GetArtifactByName returns an input artifact by its name
func (in *Inputs) GetArtifactByName(name string) *Artifact {
	return in.Artifacts.GetArtifactByName(name)
}

// GetParameterByName returns an input parameter by its name
func (in *Inputs) GetParameterByName(name string) *Parameter {
	for _, param := range in.Parameters {
		if param.Name == name {
			return &param
		}
	}
	return nil
}

// HasInputs returns whether or not there are any inputs
func (in *Inputs) HasInputs() bool {
	if len(in.Artifacts) > 0 {
		return true
	}
	if len(in.Parameters) > 0 {
		return true
	}
	return false
}

// HasOutputs returns whether or not there are any outputs
func (out *Outputs) HasOutputs() bool {
	if out.Result != nil {
		return true
	}
	if out.ExitCode != nil {
		return true
	}
	if len(out.Artifacts) > 0 {
		return true
	}
	if len(out.Parameters) > 0 {
		return true
	}
	return false
}

func (out *Outputs) GetArtifactByName(name string) *Artifact {
	if out == nil {
		return nil
	}
	return out.Artifacts.GetArtifactByName(name)
}

// GetArtifactByName retrieves an artifact by its name
func (args *Arguments) GetArtifactByName(name string) *Artifact {
	return args.Artifacts.GetArtifactByName(name)
}

// GetParameterByName retrieves a parameter by its name
func (args *Arguments) GetParameterByName(name string) *Parameter {
	for _, param := range args.Parameters {
		if param.Name == name {
			return &param
		}
	}
	return nil
}

func (a *Artifact) GetArchive() *ArchiveStrategy {
	if a == nil || a.Archive == nil {
		return &ArchiveStrategy{}
	}
	return a.Archive
}

// GetTemplateByName retrieves a defined template by its name
func (wf *Workflow) GetTemplateByName(name string) *Template {
	for _, t := range wf.Spec.Templates {
		if t.Name == name {
			return &t
		}
	}
	return nil
}

// GetResourceScope returns the template scope of workflow.
func (wf *Workflow) GetResourceScope() ResourceScope {
	return ResourceScopeLocal
}

// GetWorkflowSpec returns the Spec of a workflow.
func (wf *Workflow) GetWorkflowSpec() WorkflowSpec {
	return wf.Spec
}

// NodeID creates a deterministic node ID based on a node name
func (wf *Workflow) NodeID(name string) string {
	if name == wf.ObjectMeta.Name {
		return wf.ObjectMeta.Name
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return fmt.Sprintf("%s-%v", wf.ObjectMeta.Name, h.Sum32())
}

// GetStoredTemplate retrieves a template from stored templates of the workflow.
func (wf *Workflow) GetStoredTemplate(scope ResourceScope, resourceName string, caller TemplateReferenceHolder) *Template {
	tmplID, storageNeeded := resolveTemplateReference(scope, resourceName, caller)
	if !storageNeeded {
		// Local templates aren't stored
		return nil
	}
	if tmpl, ok := wf.Status.StoredTemplates[tmplID]; ok {
		return &tmpl
	}
	return nil
}

// SetStoredTemplate stores a new template in stored templates of the workflow.
func (wf *Workflow) SetStoredTemplate(scope ResourceScope, resourceName string, caller TemplateReferenceHolder, tmpl *Template) (bool, error) {
	tmplID, storageNeeded := resolveTemplateReference(scope, resourceName, caller)
	if !storageNeeded {
		// Don't need to store local templates
		return false, nil
	}
	if _, ok := wf.Status.StoredTemplates[tmplID]; !ok {
		if wf.Status.StoredTemplates == nil {
			wf.Status.StoredTemplates = map[string]Template{}
		}
		wf.Status.StoredTemplates[tmplID] = *tmpl
		return true, nil
	}
	return false, nil
}

// resolveTemplateReference resolves the stored template name of a given template holder on the template scope and determines
// if it should be stored
func resolveTemplateReference(callerScope ResourceScope, resourceName string, caller TemplateReferenceHolder) (string, bool) {
	tmplRef := caller.GetTemplateRef()
	if tmplRef != nil {
		// We are calling an external WorkflowTemplate or ClusterWorkflowTemplate. Template storage is needed
		// We need to determine if we're calling a WorkflowTemplate or a ClusterWorkflowTemplate
		referenceScope := ResourceScopeNamespaced
		if tmplRef.ClusterScope {
			referenceScope = ResourceScopeCluster
		}
		return fmt.Sprintf("%s/%s/%s", referenceScope, tmplRef.Name, tmplRef.Template), true
	} else if callerScope != ResourceScopeLocal {
		// Either a WorkflowTemplate or a ClusterWorkflowTemplate is calling a template inside itself. Template storage is needed
		return fmt.Sprintf("%s/%s/%s", callerScope, resourceName, caller.GetTemplateName()), true
	} else {
		// A Workflow is calling a template inside itself. Template storage is not needed
		return "", false
	}
}

// ContinueOn defines if a workflow should continue even if a task or step fails/errors.
// It can be specified if the workflow should continue when the pod errors, fails or both.
type ContinueOn struct {
	// +optional
	Error bool `json:"error,omitempty" protobuf:"varint,1,opt,name=error"`
	// +optional
	Failed bool `json:"failed,omitempty" protobuf:"varint,2,opt,name=failed"`
}

func continues(c *ContinueOn, phase NodePhase) bool {
	if c == nil {
		return false
	}
	if c.Error && phase == NodeError {
		return true
	}
	if c.Failed && phase == NodeFailed {
		return true
	}
	return false
}

// ContinuesOn returns whether the DAG should be proceeded if the task fails or errors.
func (t *DAGTask) ContinuesOn(phase NodePhase) bool {
	return continues(t.ContinueOn, phase)
}

// ContinuesOn returns whether the StepGroup should be proceeded if the task fails or errors.
func (s *WorkflowStep) ContinuesOn(phase NodePhase) bool {
	return continues(s.ContinueOn, phase)
}

type MetricType string

const (
	MetricTypeGauge     MetricType = "Gauge"
	MetricTypeHistogram MetricType = "Histogram"
	MetricTypeCounter   MetricType = "Counter"
	MetricTypeUnknown   MetricType = "Unknown"
)

// Metrics are a list of metrics emitted from a Workflow/Template
type Metrics struct {
	// Prometheus is a list of prometheus metrics to be emitted
	Prometheus []*Prometheus `json:"prometheus" protobuf:"bytes,1,rep,name=prometheus"`
}

// Prometheus is a prometheus metric to be emitted
type Prometheus struct {
	// Name is the name of the metric
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Labels is a list of metric labels
	Labels []*MetricLabel `json:"labels" protobuf:"bytes,2,rep,name=labels"`
	// Help is a string that describes the metric
	Help string `json:"help" protobuf:"bytes,3,opt,name=help"`
	// When is a conditional statement that decides when to emit the metric
	When string `json:"when" protobuf:"bytes,4,opt,name=when"`
	// Gauge is a gauge metric
	Gauge *Gauge `json:"gauge" protobuf:"bytes,5,opt,name=gauge"`
	// Histogram is a histogram metric
	Histogram *Histogram `json:"histogram" protobuf:"bytes,6,opt,name=histogram"`
	// Counter is a counter metric
	Counter *Counter `json:"counter" protobuf:"bytes,7,opt,name=counter"`
}

func (p *Prometheus) GetMetricLabels() map[string]string {
	labels := make(map[string]string)
	for _, label := range p.Labels {
		labels[label.Key] = label.Value
	}
	return labels
}

func (p *Prometheus) GetMetricType() MetricType {
	if p.Gauge != nil {
		return MetricTypeGauge
	}
	if p.Histogram != nil {
		return MetricTypeHistogram
	}
	if p.Counter != nil {
		return MetricTypeCounter
	}
	return MetricTypeUnknown
}

func (p *Prometheus) GetValueString() string {
	switch p.GetMetricType() {
	case MetricTypeGauge:
		return p.Gauge.Value
	case MetricTypeCounter:
		return p.Counter.Value
	case MetricTypeHistogram:
		return p.Histogram.Value
	default:
		return ""
	}
}

func (p *Prometheus) SetValueString(val string) {
	switch p.GetMetricType() {
	case MetricTypeGauge:
		p.Gauge.Value = val
	case MetricTypeCounter:
		p.Counter.Value = val
	case MetricTypeHistogram:
		p.Histogram.Value = val
	}
}

func (p *Prometheus) GetDesc() string {
	// This serves as a hash for the metric
	// TODO: Make sure this is what we want to use as the hash
	desc := p.Name + "{"
	for key, val := range p.GetMetricLabels() {
		desc += key + "=" + val + ","
	}
	if p.Histogram != nil {
		for _, bucket := range p.Histogram.Buckets {
			desc += "bucket=" + fmt.Sprint(bucket) + ","
		}
	}
	desc += "}"
	return desc
}

func (p *Prometheus) IsRealtime() bool {
	return p.GetMetricType() == MetricTypeGauge && p.Gauge.Realtime != nil && *p.Gauge.Realtime
}

// MetricLabel is a single label for a prometheus metric
type MetricLabel struct {
	Key   string `json:"key" protobuf:"bytes,1,opt,name=key"`
	Value string `json:"value" protobuf:"bytes,2,opt,name=value"`
}

// Gauge is a Gauge prometheus metric
type Gauge struct {
	// Value is the value of the metric
	Value string `json:"value" protobuf:"bytes,1,opt,name=value"`
	// Realtime emits this metric in real time if applicable
	Realtime *bool `json:"realtime" protobuf:"varint,2,opt,name=realtime"`
}

// Histogram is a Histogram prometheus metric
type Histogram struct {
	// Value is the value of the metric
	Value string `json:"value" protobuf:"bytes,3,opt,name=value"`
	// Buckets is a list of bucket divisors for the histogram
	Buckets []float64 `json:"buckets" protobuf:"fixed64,4,rep,name=buckets"`
}

// Counter is a Counter prometheus metric
type Counter struct {
	// Value is the value of the metric
	Value string `json:"value" protobuf:"bytes,1,opt,name=value"`
}
