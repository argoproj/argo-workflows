package v1alpha1

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Pool runs a container or script template's tasks on a pool of long-lived
// worker pods, one task at a time per worker pod, instead of a pod per task.
type Pool struct {
	// Name of the pool. Templates with the same pool name and the same pod
	// settings share a pool. Defaults to `<workflowtemplate>-<template>` for a
	// template from a WorkflowTemplate or ClusterWorkflowTemplate, otherwise to
	// the template's name. It must be a DNS label of at most 35 characters. It
	// may hold a variable, such as `{{inputs.parameters.pool}}`, and is then
	// checked once substituted.
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// InitialReplicas is the number of worker pods a new pool starts with.
	// It applies only when the pool is created. Defaults to 1.
	// +kubebuilder:validation:Minimum=1
	// +optional
	InitialReplicas *int32 `json:"initialReplicas,omitempty" protobuf:"varint,2,opt,name=initialReplicas"`

	// RetentionSeconds is how long the pool is kept once no task uses it.
	// The most recently dispatched task's value applies. Defaults to the
	// controller's hot workers retention.
	// +kubebuilder:validation:Minimum=0
	// +optional
	RetentionSeconds *int64 `json:"retentionSeconds,omitempty" protobuf:"varint,3,opt,name=retentionSeconds"`
}

// PoolSpec is everything a pool's worker pods are built from, resolved by the
// controller when it dispatches a pooled task. The pool's hash is computed
// from it, excluding InitialReplicas and RetentionSeconds.
type PoolSpec struct {
	// Name is the resolved pool name.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

	// InitialReplicas is the number of worker pods a new pool starts with.
	InitialReplicas int32 `json:"initialReplicas" protobuf:"varint,2,opt,name=initialReplicas"`

	// RetentionSeconds is how long the pool is kept once no task uses it.
	RetentionSeconds int64 `json:"retentionSeconds" protobuf:"varint,3,opt,name=retentionSeconds"`

	// Template holds the merged template's fields that are the same for every
	// task in the pool. Per-task fields are cleared.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	Template *Template `json:"template,omitempty" protobuf:"bytes,4,opt,name=template"`

	// WorkflowSpec holds the workflow-level fields that feed the pool's pods.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	WorkflowSpec *WorkflowSpec `json:"workflowSpec,omitempty" protobuf:"bytes,5,opt,name=workflowSpec"`

	// ServiceAccountName is the resolved service account the pool's pods run as.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty" protobuf:"bytes,6,opt,name=serviceAccountName"`

	// AutomountServiceAccountToken is the resolved automountServiceAccountToken.
	// +optional
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty" protobuf:"varint,7,opt,name=automountServiceAccountToken"`

	// ExecutorServiceAccountTokenSecretName is the Secret holding the executor
	// service account's token, set when the executor runs as its own service
	// account. It is mounted only in the worker container.
	// +optional
	ExecutorServiceAccountTokenSecretName string `json:"executorServiceAccountTokenSecretName,omitempty" protobuf:"bytes,8,opt,name=executorServiceAccountTokenSecretName"`

	// ArtifactRepository is the selected artifact repository, with its key
	// format as written.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	ArtifactRepository *ArtifactRepository `json:"artifactRepository,omitempty" protobuf:"bytes,9,opt,name=artifactRepository"`

	// ArtifactSecrets are the secrets the artifact repository and the
	// template's input, output and data-source artifacts read. They are
	// mounted only in the worker container.
	// +listType=atomic
	// +optional
	ArtifactSecrets []apiv1.SecretKeySelector `json:"artifactSecrets,omitempty" protobuf:"bytes,10,rep,name=artifactSecrets"`

	// ArtifactPlugins are the artifact driver plugins the template's artifacts
	// use, run as sidecars of the pool's pods.
	// +listType=atomic
	// +optional
	ArtifactPlugins []PoolArtifactPlugin `json:"artifactPlugins,omitempty" protobuf:"bytes,11,rep,name=artifactPlugins"`

	// InstanceID is the controller instance ID.
	// +optional
	InstanceID string `json:"instanceID,omitempty" protobuf:"bytes,12,opt,name=instanceID"`

	// ControllerVersion is the version of the controller that dispatched the task.
	ControllerVersion string `json:"controllerVersion" protobuf:"bytes,13,opt,name=controllerVersion"`

	// Executor is the effective executor container configuration.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	Executor *apiv1.Container `json:"executor,omitempty" protobuf:"bytes,14,opt,name=executor"`

	// MainContainer is the controller's main container defaults.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	MainContainer *apiv1.Container `json:"mainContainer,omitempty" protobuf:"bytes,15,opt,name=mainContainer"`

	// ExecutorImage is the effective argoexec image.
	ExecutorImage string `json:"executorImage" protobuf:"bytes,16,opt,name=executorImage"`

	// ExecutorImagePullPolicy is the effective argoexec image pull policy.
	// +optional
	ExecutorImagePullPolicy apiv1.PullPolicy `json:"executorImagePullPolicy,omitempty" protobuf:"bytes,17,opt,name=executorImagePullPolicy,casttype=k8s.io/api/core/v1.PullPolicy"`

	// ExecutorLogArgs are the log flags passed to every argoexec container.
	// +listType=atomic
	// +optional
	ExecutorLogArgs []string `json:"executorLogArgs,omitempty" protobuf:"bytes,18,rep,name=executorLogArgs"`

	// KubeConfig is the kubeconfig the argoexec containers use, when the
	// controller is configured with one.
	// +optional
	KubeConfig *PoolKubeConfig `json:"kubeConfig,omitempty" protobuf:"bytes,19,opt,name=kubeConfig"`

	// InitlessPod selects the init-less pod layout.
	// +optional
	InitlessPod bool `json:"initlessPod,omitempty" protobuf:"varint,20,opt,name=initlessPod"`

	// TerminationGracePeriodSeconds is the worker pods' termination grace period.
	TerminationGracePeriodSeconds int64 `json:"terminationGracePeriodSeconds" protobuf:"varint,21,opt,name=terminationGracePeriodSeconds"`

	// ReleaseReserveSeconds is how long before the end of the grace period a
	// worker stops a running task so it can release it.
	ReleaseReserveSeconds int64 `json:"releaseReserveSeconds" protobuf:"varint,22,opt,name=releaseReserveSeconds"`
}

// PoolArtifactPlugin is an artifact driver plugin run as a sidecar of a
// pool's pods.
type PoolArtifactPlugin struct {
	// Name of the artifact driver plugin.
	Name ArtifactPluginName `json:"name" protobuf:"bytes,1,opt,name=name,casttype=ArtifactPluginName"`

	// Image of the artifact driver plugin.
	Image string `json:"image" protobuf:"bytes,2,opt,name=image"`

	// ConnectionTimeoutSeconds is the timeout for connecting to the plugin.
	// +optional
	ConnectionTimeoutSeconds int32 `json:"connectionTimeoutSeconds,omitempty" protobuf:"varint,3,opt,name=connectionTimeoutSeconds"`

	// Command is the image's entrypoint, which the plugin's sidecar runs,
	// looked up at dispatch as a pod's plugin containers look it up.
	// +listType=atomic
	// +optional
	Command []string `json:"command,omitempty" protobuf:"bytes,4,rep,name=command"`
}

// PoolKubeConfig is the kubeconfig Secret the argoexec containers of a pool's
// pods mount.
type PoolKubeConfig struct {
	// SecretName of the kubeconfig Secret.
	SecretName string `json:"secretName" protobuf:"bytes,1,opt,name=secretName"`

	// SecretKey of the kubeconfig in the Secret.
	SecretKey string `json:"secretKey" protobuf:"bytes,2,opt,name=secretKey"`

	// VolumeName of the kubeconfig volume.
	VolumeName string `json:"volumeName" protobuf:"bytes,3,opt,name=volumeName"`

	// MountPath of the kubeconfig.
	MountPath string `json:"mountPath" protobuf:"bytes,4,opt,name=mountPath"`
}

// TaskDispatch is the per-task part of a pooled task that a pod would get
// from its spec and environment.
type TaskDispatch struct {
	// TaskName is the task's `{{node.taskName}}`: the name its pod would have.
	TaskName string `json:"taskName" protobuf:"bytes,1,opt,name=taskName"`

	// IncludeScriptOutput captures the task's stdout as its result.
	// +optional
	IncludeScriptOutput bool `json:"includeScriptOutput,omitempty" protobuf:"varint,3,opt,name=includeScriptOutput"`

	// InputArtifactPlugins are the artifact driver plugins the task's input
	// artifacts use.
	// +listType=atomic
	// +optional
	InputArtifactPlugins []ArtifactPluginName `json:"inputArtifactPlugins,omitempty" protobuf:"bytes,5,rep,name=inputArtifactPlugins,casttype=ArtifactPluginName"`

	// Command is the command to run when the template's container has no
	// command: the image's entrypoint, followed by the image's cmd when the
	// task has no arguments of its own.
	// +listType=atomic
	// +optional
	Command []string `json:"command,omitempty" protobuf:"bytes,7,rep,name=command"`

	// Deadline is the task's absolute deadline: the earliest of the workflow
	// deadline (except for exit handler tasks), the retry strategy's
	// maxDuration, and the node's start plus the template's timeout.
	// +optional
	Deadline *metav1.Time `json:"deadline,omitempty" protobuf:"bytes,8,opt,name=deadline"`

	// ActiveDeadlineSeconds limits how long the task may run once claimed.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty" protobuf:"varint,10,opt,name=activeDeadlineSeconds"`

	// TraceContext is the W3C trace context the task's spans continue.
	// +optional
	TraceContext map[string]string `json:"traceContext,omitempty" protobuf:"bytes,11,rep,name=traceContext"`

	// ProgressPatchTickDuration is how often the task's progress is reported.
	// Progress is not reported when this or ProgressFileTickDuration is unset.
	// +optional
	ProgressPatchTickDuration *metav1.Duration `json:"progressPatchTickDuration,omitempty" protobuf:"bytes,12,opt,name=progressPatchTickDuration"`

	// ProgressFileTickDuration is how often the task's progress file is read.
	// +optional
	ProgressFileTickDuration *metav1.Duration `json:"progressFileTickDuration,omitempty" protobuf:"bytes,13,opt,name=progressFileTickDuration"`
}

// PooledTaskState is where a pooled task is in its hand-off to a worker. It is
// derived from the ledger's conditions (PooledTaskStatus.State), never stored.
type PooledTaskState string

const (
	// PooledTaskEnqueued is a task waiting for a worker, including one a
	// worker released.
	PooledTaskEnqueued PooledTaskState = "Enqueued"
	// PooledTaskClaimed is a task a worker has claimed.
	PooledTaskClaimed PooledTaskState = "Claimed"
	// PooledTaskInterrupted is a claimed task the controller has ended and is
	// waiting to see stopped.
	PooledTaskInterrupted PooledTaskState = "Interrupted"
)

// The condition types and reasons of a pooled task's ledger.
const (
	// PooledTaskConditionClaimed is True while a worker holds the task, with
	// the claimant pod in its message and the claim's creation time as its
	// lastTransitionTime. It is False with reason PooledTaskReasonEnqueued
	// before any claim, and with PooledTaskReasonReleased, naming the last
	// claimant, after a release.
	PooledTaskConditionClaimed = "Claimed"
	// PooledTaskConditionInterrupted is True once the controller has
	// interrupted the claimed task.
	PooledTaskConditionInterrupted = "Interrupted"

	PooledTaskReasonEnqueued    = "Enqueued"
	PooledTaskReasonClaimed     = "Claimed"
	PooledTaskReasonReleased    = "Released"
	PooledTaskReasonInterrupted = "Interrupted"
)

// PooledTaskStatus is the controller's record of a pooled node's task.
type PooledTaskStatus struct {
	// PoolName is the name of the pool the task was dispatched to.
	PoolName string `json:"poolName" protobuf:"bytes,1,opt,name=poolName"`

	// PoolHash is the hash of the pool the task was dispatched to.
	PoolHash string `json:"poolHash" protobuf:"bytes,2,opt,name=poolHash"`

	// TaskName is the task's `{{node.taskName}}`.
	TaskName string `json:"taskName" protobuf:"bytes,3,opt,name=taskName"`

	// TaskSetUID is the UID of the task's WorkflowTaskSet. Only results that
	// carry it are applied to the node.
	TaskSetUID types.UID `json:"taskSetUID" protobuf:"bytes,5,opt,name=taskSetUID,casttype=k8s.io/apimachinery/pkg/types.UID"`

	// EnqueuedAt is when the task's WorkflowTaskSet was created or adopted.
	EnqueuedAt metav1.Time `json:"enqueuedAt" protobuf:"bytes,6,opt,name=enqueuedAt"`

	// Deadline is the task's absolute deadline, as dispatched.
	// +optional
	Deadline *metav1.Time `json:"deadline,omitempty" protobuf:"bytes,7,opt,name=deadline"`

	// ActiveDeadlineSeconds limits how long the task may run once claimed, as dispatched.
	// +optional
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty" protobuf:"varint,8,opt,name=activeDeadlineSeconds"`

	// ClaimUID is the UID of the WorkflowTaskResult the claiming worker created.
	// +optional
	ClaimUID types.UID `json:"claimUID,omitempty" protobuf:"bytes,9,opt,name=claimUID,casttype=k8s.io/apimachinery/pkg/types.UID"`

	// WorkerPodName is the name of the claiming worker pod.
	// +optional
	WorkerPodName string `json:"workerPodName,omitempty" protobuf:"bytes,11,opt,name=workerPodName"`

	// WorkerPodUID is the UID of the claiming worker pod.
	// +optional
	WorkerPodUID types.UID `json:"workerPodUID,omitempty" protobuf:"bytes,12,opt,name=workerPodUID,casttype=k8s.io/apimachinery/pkg/types.UID"`

	// WorkerRestartCount is the claiming pod's worker container restart count
	// when the claim was first seen.
	// +optional
	WorkerRestartCount int32 `json:"workerRestartCount,omitempty" protobuf:"varint,13,opt,name=workerRestartCount"`

	// Conditions record the task's hand-off to a worker: Claimed and
	// Interrupted. Its state is derived from them (State). Change them only
	// through Enqueue, Claim, Release and Interrupt.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,15,rep,name=conditions"`

	// Releases is how many times a worker has released the task.
	// +optional
	Releases int32 `json:"releases,omitempty" protobuf:"varint,16,opt,name=releases"`
}

// State is where the task is in its hand-off, from its conditions:
// Interrupted once the controller has interrupted it, Claimed while a worker
// holds it, and Enqueued otherwise.
func (s *PooledTaskStatus) State() PooledTaskState {
	switch {
	case meta.IsStatusConditionTrue(s.Conditions, PooledTaskConditionInterrupted):
		return PooledTaskInterrupted
	case meta.IsStatusConditionTrue(s.Conditions, PooledTaskConditionClaimed):
		return PooledTaskClaimed
	default:
		return PooledTaskEnqueued
	}
}

// ClaimedAt is the current claim's creation time, on the API server's clock,
// or nil if no worker holds the task.
func (s *PooledTaskStatus) ClaimedAt() *metav1.Time {
	if c := meta.FindStatusCondition(s.Conditions, PooledTaskConditionClaimed); c != nil && c.Status == metav1.ConditionTrue {
		return &c.LastTransitionTime
	}
	return nil
}

// InterruptedAt is when the controller interrupted the task, or nil if it
// hasn't.
func (s *PooledTaskStatus) InterruptedAt() *metav1.Time {
	if c := meta.FindStatusCondition(s.Conditions, PooledTaskConditionInterrupted); c != nil && c.Status == metav1.ConditionTrue {
		return &c.LastTransitionTime
	}
	return nil
}

// Released reports whether a worker released the task and none has claimed
// it since.
func (s *PooledTaskStatus) Released() bool {
	c := meta.FindStatusCondition(s.Conditions, PooledTaskConditionClaimed)
	return c != nil && c.Status == metav1.ConditionFalse && c.Reason == PooledTaskReasonReleased
}

// Enqueue records the task as waiting for its first claim, its TaskSet
// created or adopted at at.
func (s *PooledTaskStatus) Enqueue(at metav1.Time) {
	s.EnqueuedAt = at
	s.setCondition(PooledTaskConditionClaimed, metav1.ConditionFalse, PooledTaskReasonEnqueued, "", at)
}

// Claim records a worker's claim: its result's UID and creation time, and the
// claimant pod with its worker container's restart count. A claim replaces any
// claim before it. It returns false, changing nothing, once the task is
// interrupted.
func (s *PooledTaskStatus) Claim(claimUID types.UID, claimedAt metav1.Time, podName string, podUID types.UID, workerRestartCount int32) bool {
	if s.State() == PooledTaskInterrupted {
		return false
	}
	s.ClaimUID, s.WorkerPodName, s.WorkerPodUID, s.WorkerRestartCount = claimUID, podName, podUID, workerRestartCount
	s.setCondition(PooledTaskConditionClaimed, metav1.ConditionTrue, PooledTaskReasonClaimed, podName, claimedAt)
	return true
}

// Release records that the claimant released the task at at: it is waiting
// for a worker again, the claimant is cleared, and one more release is
// counted. It returns false, changing nothing, unless the task is claimed.
func (s *PooledTaskStatus) Release(at metav1.Time) bool {
	if s.State() != PooledTaskClaimed {
		return false
	}
	s.setCondition(PooledTaskConditionClaimed, metav1.ConditionFalse, PooledTaskReasonReleased, s.WorkerPodName, at)
	s.ClaimUID, s.WorkerPodName, s.WorkerPodUID, s.WorkerRestartCount = "", "", "", 0
	s.Releases++
	return true
}

// Interrupt records that the controller interrupted the claimed task at at.
// An interrupted task never goes back to claimed or enqueued. It returns
// false, changing nothing, unless the task is claimed.
func (s *PooledTaskStatus) Interrupt(at metav1.Time) bool {
	if s.State() != PooledTaskClaimed {
		return false
	}
	s.setCondition(PooledTaskConditionInterrupted, metav1.ConditionTrue, PooledTaskReasonInterrupted, "", at)
	return true
}

// setCondition replaces the condition of type t. Unlike
// meta.SetStatusCondition, it always takes at as the transition time, so a
// claim replacing a claim records the new claim's time.
func (s *PooledTaskStatus) setCondition(t string, status metav1.ConditionStatus, reason, message string, at metav1.Time) {
	c := metav1.Condition{Type: t, Status: status, Reason: reason, Message: message, LastTransitionTime: at}
	for i := range s.Conditions {
		if s.Conditions[i].Type == t {
			s.Conditions[i] = c
			return
		}
	}
	s.Conditions = append(s.Conditions, c)
}
