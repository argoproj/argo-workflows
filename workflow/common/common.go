package common

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow"
)

const (
	// Container names used in the workflow pod
	MainContainerName = "main"
	InitContainerName = "init"
	WaitContainerName = "wait"

	// AnnotationKeyDefaultContainer is the annotation that specify container that will be used by default in case of kubectl commands for example
	AnnotationKeyDefaultContainer = "kubectl.kubernetes.io/default-container"

	// AnnotationKeyServiceAccountTokenName is used to name the secret that containers the service account token name.
	// It is intentionally named similar to ` `kubernetes.io/service-account.name`.
	AnnotationKeyServiceAccountTokenName = workflow.WorkflowFullName + "/service-account-token.name"

	// AnnotationKeyNodeID is the ID of the node.
	// Historically, the pod name was the same as the node ID.
	// Therefore, if it does not exist, then the node ID is the pod name.
	AnnotationKeyNodeID = workflow.WorkflowFullName + "/node-id"
	// AnnotationKeyNodeName is the pod metadata annotation key containing the workflow node name
	AnnotationKeyNodeName = workflow.WorkflowFullName + "/node-name"
	// AnnotationKeyNodeType is the node's type
	AnnotationKeyNodeType = workflow.WorkflowFullName + "/node-type"
	// AnnotationKeyNodeStartTime is the node's start timestamp.
	AnnotationKeyNodeStartTime = workflow.WorkflowFullName + "/node-start-time"

	// AnnotationKeyRBACRule is a rule to match the claims
	AnnotationKeyRBACRule           = workflow.WorkflowFullName + "/rbac-rule"
	AnnotationKeyRBACRulePrecedence = workflow.WorkflowFullName + "/rbac-rule-precedence"

	// AnnotationKeyCronWfScheduledTime is the workflow metadata annotation key containing the time when the workflow
	// was scheduled to run by CronWorkflow.
	AnnotationKeyCronWfScheduledTime = workflow.WorkflowFullName + "/scheduled-time"

	// AnnotationKeyWorkflowName is the name of the workflow
	AnnotationKeyWorkflowName = workflow.WorkflowFullName + "/workflow-name"
	// AnnotationKeyWorkflowUID is the uid of the workflow
	AnnotationKeyWorkflowUID = workflow.WorkflowFullName + "/workflow-uid"

	// AnnotationKeyPodNameVersion stores the pod naming convention version
	AnnotationKeyPodNameVersion = workflow.WorkflowFullName + "/pod-name-format"

	// AnnotationKeyProgress is N/M progress for the node
	AnnotationKeyProgress = workflow.WorkflowFullName + "/progress"

	// AnnotationKeyArtifactGCStrategy is listed as an annotation on the Artifact GC Pod to identify
	// the strategy whose artifacts are being deleted
	AnnotationKeyArtifactGCStrategy = workflow.WorkflowFullName + "/artifact-gc-strategy"

	// AnnotationKeyLastSeenVersion stores the last seen version of the workflow when it was last successfully processed by the controller
	AnnotationKeyLastSeenVersion = workflow.WorkflowFullName + "/last-seen-version"

	// LabelParallelismLimit is a label applied on namespace objects to control the per namespace parallelism.
	LabelParallelismLimit = workflow.WorkflowFullName + "/parallelism-limit"

	// AnnotationKeyPodGCStrategy is listed as an annotation on the Pod
	// the strategy for the pod, in case the pod is orphaned from its workflow
	AnnotationKeyPodGCStrategy = workflow.WorkflowFullName + "/pod-gc-strategy"

	// LabelKeyControllerInstanceID is the label the controller will carry forward to workflows/pod labels
	// for the purposes of workflow segregation
	LabelKeyControllerInstanceID = workflow.WorkflowFullName + "/controller-instanceid"
	// Who created this workflow.
	LabelKeyCreator                  = workflow.WorkflowFullName + "/creator"
	LabelKeyCreatorEmail             = workflow.WorkflowFullName + "/creator-email"
	LabelKeyCreatorPreferredUsername = workflow.WorkflowFullName + "/creator-preferred-username"
	// Who action on this workflow.
	LabelKeyActor                  = workflow.WorkflowFullName + "/actor"
	LabelKeyActorEmail             = workflow.WorkflowFullName + "/actor-email"
	LabelKeyActorPreferredUsername = workflow.WorkflowFullName + "/actor-preferred-username"
	LabelKeyAction                 = workflow.WorkflowFullName + "/action"
	// LabelKeyCompleted is the metadata label applied on workflows and workflow pods to indicates if resource is completed
	// Workflows and pods with a completed=true label will be ignored by the controller.
	// See also `LabelKeyWorkflowArchivingStatus`.
	LabelKeyCompleted = workflow.WorkflowFullName + "/completed"
	// LabelKeyWorkflowArchivingStatus indicates if a workflow needs archiving or not:
	// * `` - does not need archiving ... yet
	// * `Pending` - pending archiving
	// * `Archived` - has been archived and has live manifest
	// * `Persisted` - has been archived and retrieved from db
	// See also `LabelKeyCompleted`.
	LabelKeyWorkflowArchivingStatus = workflow.WorkflowFullName + "/workflow-archiving-status"
	// LabelKeyWorkflow is the pod metadata label to indicate the associated workflow name
	LabelKeyWorkflow = workflow.WorkflowFullName + "/workflow"
	// LabelKeyComponent determines what component within a workflow, intentionally similar to app.kubernetes.io/component.
	// See https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
	LabelKeyComponent = workflow.WorkflowFullName + "/component"
	// LabelKeyPhase is a label applied to workflows to indicate the current phase of the workflow (for filtering purposes)
	LabelKeyPhase = workflow.WorkflowFullName + "/phase"
	// LabelKeyPreviousWorkflowName is a label applied to resubmitted workflows
	LabelKeyPreviousWorkflowName = workflow.WorkflowFullName + "/resubmitted-from-workflow"
	// LabelKeyCronWorkflow is a label applied to Workflows that are started by a CronWorkflow
	LabelKeyCronWorkflow = workflow.WorkflowFullName + "/cron-workflow"
	// LabelKeyWorkflowTemplate is a label applied to Workflows that are submitted from Workflowtemplate
	LabelKeyWorkflowTemplate = workflow.WorkflowFullName + "/workflow-template"
	// LabelKeyWorkflowEventBinding is a label applied to Workflows that are submitted from a WorkflowEventBinding
	LabelKeyWorkflowEventBinding = workflow.WorkflowFullName + "/workflow-event-binding"
	// LabelKeyClusterWorkflowTemplate is a label applied to Workflows that are submitted from ClusterWorkflowtemplate
	LabelKeyClusterWorkflowTemplate = workflow.WorkflowFullName + "/cluster-workflow-template"
	// LabelKeyOnExit is a label applied to Pods that are run from onExit nodes, so that they are not shut down when stopping a Workflow
	LabelKeyOnExit = workflow.WorkflowFullName + "/on-exit"
	// LabelKeyArtifactGCPodHash is a label applied to WorkflowTaskSets used by the Artifact Garbage Collection Pod
	LabelKeyArtifactGCPodHash = workflow.WorkflowFullName + "/artifact-gc-pod"
	// LabelKeyReportOutputsCompleted is a label applied to WorkflowTaskResults indicating whether all the outputs have been reported.
	LabelKeyReportOutputsCompleted = workflow.WorkflowFullName + "/report-outputs-completed"

	// LabelKeyCronWorkflowCompleted is a label applied to the cron workflow when the configured stopping condition is achieved
	LabelKeyCronWorkflowCompleted = workflow.CronWorkflowFullName + "/completed"

	// LabelKeyCronWorkflowBackfill is a label applied to the cron workflow when the workflow is created by backfill
	LabelKeyCronWorkflowBackfill = workflow.WorkflowFullName + "/backfill"

	// ExecutorArtifactBaseDir is the base directory in the init container in which artifacts will be copied to.
	// Each artifact will be named according to its input name (e.g: /argo/inputs/artifacts/CODE)
	ExecutorArtifactBaseDir = "/argo/inputs/artifacts"

	// ExecutorMainFilesystemDir is a path made available to the init/wait containers such that they
	// can access the same volume mounts used in the main container. This is used for the purposes
	// of artifact loading (when there is overlapping paths between artifacts and volume mounts),
	// as well as artifact collection by the wait container.
	ExecutorMainFilesystemDir = "/mainctrfs"

	// ExecutorStagingEmptyDir is the path of the emptydir which is used as a staging area to transfer a file between init/main container for script/resource templates
	ExecutorStagingEmptyDir = "/argo/staging"
	// ExecutorScriptSourcePath is the path which init will write the script source file to for script templates
	ExecutorScriptSourcePath = "/argo/staging/script"
	// ExecutorResourceManifestPath is the path which init will write the manifest file to for resource templates
	ExecutorResourceManifestPath = "/tmp/manifest.yaml"

	// Various environment variables containing pod information exposed to the executor container(s)

	// EnvVarArtifactGCPodHash is applied as a Label on the WorkflowTaskSets read by the Artifact GC Pod, so that the Pod can find them
	EnvVarArtifactGCPodHash = "ARGO_ARTIFACT_POD_NAME"
	// EnvVarPodName contains the name of the pod (currently unused)
	EnvVarPodName = "ARGO_POD_NAME"
	// EnvVarPodUID is the workflow's UID
	EnvVarPodUID = "ARGO_POD_UID"
	// EnvVarInstanceID is the instance ID
	EnvVarInstanceID = "ARGO_INSTANCE_ID"
	// EnvVarWorkflowName is the name of the workflow for which the an agent is responsible for
	EnvVarWorkflowName = "ARGO_WORKFLOW_NAME"
	// EnvVarWorkflowUID is the workflow UUID
	EnvVarWorkflowUID = "ARGO_WORKFLOW_UID"
	// EnvVarNodeID is the node ID of the node.
	EnvVarNodeID = "ARGO_NODE_ID"
	// EnvVarPluginAddresses is a list of plugin addresses
	EnvVarPluginAddresses = "ARGO_PLUGIN_ADDRESSES"
	// EnvVarPluginNames is a list of plugin names
	EnvVarPluginNames = "ARGO_PLUGIN_NAMES"
	// EnvVarContainerName container the container's name for the current pod
	EnvVarContainerName = "ARGO_CONTAINER_NAME"
	// EnvVarDeadline is the deadline for the pod
	EnvVarDeadline = "ARGO_DEADLINE"
	// EnvVarTerminationGracePeriodSeconds is pod.spec.terminationGracePeriodSeconds
	EnvVarTerminationGracePeriodSeconds = "ARGO_TERMINATION_GRACE_PERIOD_SECONDS"
	// EnvVarIncludeScriptOutput capture the stdout and stderr
	EnvVarIncludeScriptOutput = "ARGO_INCLUDE_SCRIPT_OUTPUT"
	// EnvVarTemplate is the template
	EnvVarTemplate = "ARGO_TEMPLATE"
	// EnvVarArgoTrace is used enable tracing statements in Argo components
	EnvVarArgoTrace = "ARGO_TRACE"
	// EnvVarProgressPatchTickDuration sets the tick duration for patching pod annotations upon progress changes.
	// Setting this or EnvVarProgressFileTickDuration to 0 will disable monitoring progress.
	EnvVarProgressPatchTickDuration = "ARGO_PROGRESS_PATCH_TICK_DURATION"
	// EnvVarProgressFileTickDuration sets the tick duration for reading & parsing the progress file.
	// Setting this or EnvVarProgressPatchTickDuration to 0 will disable monitoring progress.
	EnvVarProgressFileTickDuration = "ARGO_PROGRESS_FILE_TICK_DURATION"
	// EnvVarProgressFile is the file watched for reporting progress
	EnvVarProgressFile = "ARGO_PROGRESS_FILE"
	// EnvVarDefaultRequeueTime is the default requeue time for Workflow Informers. For more info, see rate_limiters.go
	EnvVarDefaultRequeueTime = "DEFAULT_REQUEUE_TIME"
	// EnvVarPodStatusCaptureFinalizer is used to prevent pod garbage collected before argo captures its exit status
	EnvVarPodStatusCaptureFinalizer = "ARGO_POD_STATUS_CAPTURE_FINALIZER"
	// EnvAgentTaskWorkers is the number of task workers for the agent pod
	EnvAgentTaskWorkers = "ARGO_AGENT_TASK_WORKERS"
	// EnvAgentPatchRate is the rate that the Argo Agent will patch the Workflow TaskSet
	EnvAgentPatchRate = "ARGO_AGENT_PATCH_RATE"

	// Finalizer to block deletion of the workflow if deletion of artifacts fail for some reason.
	FinalizerArtifactGC = workflow.WorkflowFullName + "/artifact-gc"

	// Finalizer blocks the deletion of pods until the controller captures their status.
	FinalizerPodStatus = workflow.WorkflowFullName + "/status"

	// Variables that are added to the scope during template execution and can be referenced using {{}} syntax

	// GlobalVarWorkflowName is a global workflow variable referencing the workflow's metadata.name field
	GlobalVarWorkflowName = "workflow.name"
	// GlobalVarWorkflowNamespace is a global workflow variable referencing the workflow's metadata.namespace field
	GlobalVarWorkflowNamespace = "workflow.namespace"
	// GlobalVarWorkflowMainEntrypoint is a global workflow variable referencing the workflow's top level entrypoint name
	GlobalVarWorkflowMainEntrypoint = "workflow.mainEntrypoint"
	// GlobalVarWorkflowServiceAccountName is a global workflow variable referencing the workflow's spec.serviceAccountName field
	GlobalVarWorkflowServiceAccountName = "workflow.serviceAccountName"
	// GlobalVarWorkflowUID is a global workflow variable referencing the workflow's metadata.uid field
	GlobalVarWorkflowUID = "workflow.uid"
	// GlobalVarWorkflowStatus is a global workflow variable referencing the workflow's status.phase field
	GlobalVarWorkflowStatus = "workflow.status"
	// GlobalVarWorkflowCreationTimestamp is the workflow variable referencing the workflow's metadata.creationTimestamp field
	GlobalVarWorkflowCreationTimestamp = "workflow.creationTimestamp"
	// GlobalVarWorkflowPriority is the workflow variable referencing the workflow's priority field
	GlobalVarWorkflowPriority = "workflow.priority"
	// GlobalVarWorkflowFailures is a global variable of a JSON map referencing the workflow's failed nodes
	GlobalVarWorkflowFailures = "workflow.failures"
	// GlobalVarWorkflowDuration is the current duration of this workflow
	GlobalVarWorkflowDuration = "workflow.duration"
	// GlobalVarWorkflowAnnotations is a JSON string containing all workflow annotations - which will be deprecated in favor of GlobalVarWorkflowAnnotationsJSON
	GlobalVarWorkflowAnnotations = "workflow.annotations"
	// GlobalVarWorkflowAnnotationsJSON is a JSON string containing all workflow annotations
	GlobalVarWorkflowAnnotationsJSON = "workflow.annotations.json"
	// GlobalVarWorkflowLabels is a JSON string containing all workflow labels - which will be deprecated in favor of GlobalVarWorkflowLabelsJSON
	GlobalVarWorkflowLabels = "workflow.labels"
	// GlobalVarWorkflowLabelsJSON is a JSON string containing all workflow labels
	GlobalVarWorkflowLabelsJSON = "workflow.labels.json"
	// GlobalVarWorkflowParameters is a JSON string containing all workflow parameters - which will be deprecated in favor of GlobalVarWorkflowParametersJSON
	GlobalVarWorkflowParameters = "workflow.parameters"
	// GlobalVarWorkflowParametersJSON is a JSON string containing all workflow parameters
	GlobalVarWorkflowParametersJSON = "workflow.parameters.json"
	// GlobalVarWorkflowCronScheduleTime is the scheduled timestamp of a Workflow started by a CronWorkflow
	GlobalVarWorkflowCronScheduleTime = "workflow.scheduledTime"

	// LabelKeyConfigMapType is the label key for the type of configmap.
	LabelKeyConfigMapType = "workflows.argoproj.io/configmap-type"
	// LabelValueTypeConfigMapCache is a key for configmaps that are memoization cache.
	LabelValueTypeConfigMapCache = "Cache"
	// LabelValueTypeConfigMapParameter is a key for configmaps that contains parameter values.
	LabelValueTypeConfigMapParameter = "Parameter"
	// LabelValueTypeConfigMapExecutorPlugin is a key for configmaps that contains an executor plugin.
	LabelValueTypeConfigMapExecutorPlugin = "ExecutorPlugin"

	// LocalVarPodName is a step level variable that references the name of the pod
	LocalVarPodName = "pod.name"
	// LocalVarRetries is a step level variable that references the retries number if retryStrategy is specified
	LocalVarRetries = "retries"
	// LocalVarDuration is a step level variable (currently only available in metric emission) that tracks the duration of the step
	LocalVarDuration = "duration"
	// LocalVarStatus is a step level variable (currently only available in metric emission) that tracks the duration of the step
	LocalVarStatus = "status"
	// LocalVarResourcesDuration is a step level variable (currently only available in metric emission) that tracks the resources duration of the step
	LocalVarResourcesDuration = "resourcesDuration"
	// LocalVarExitCode is a step level variable (currently only available in metric emission) that tracks the step's exit code
	LocalVarExitCode = "exitCode"

	// LocalVarRetriesLastExitCode is a variable that references information about the last retry's exit code
	LocalVarRetriesLastExitCode = "lastRetry.exitCode"
	// LocalVarRetriesLastStatus is a variable that references information about the last retry's status
	LocalVarRetriesLastStatus = "lastRetry.status"
	// LocalVarRetriesLastDuration is a variable that references information about the last retry's duration, in seconds
	LocalVarRetriesLastDuration = "lastRetry.duration"
	// LocalVarRetriesLastMessage is a variable that references information about the last retry's failure message
	LocalVarRetriesLastMessage = "lastRetry.message"

	KubeConfigDefaultMountPath    = "/kube/config"
	KubeConfigDefaultVolumeName   = "kubeconfig"
	ServiceAccountTokenMountPath  = "/var/run/secrets/kubernetes.io/serviceaccount" //nolint:gosec
	ServiceAccountTokenVolumeName = "exec-sa-token"                                 //nolint:gosec
	SecretVolMountPath            = "/argo/secret"
	EnvConfigMountPath            = "/argo/config"
	EnvVarTemplateOffloaded       = "offloaded"

	// CACertificatesVolumeMountName is the name of the secret that contains the CA certificates.
	CACertificatesVolumeMountName = "argo-workflows-agent-ca-certificates"

	// VarRunArgoPath is the standard path for the shared volume
	VarRunArgoPath = "/var/run/argo"

	// ArgoProgressPath defines the path to a file used for self reporting progress
	ArgoProgressPath = VarRunArgoPath + "/progress"

	ConfigMapName = "workflow-controller-configmap"
)

// AnnotationKeyKillCmd specifies the command to use to kill to container, useful for injected sidecars
var AnnotationKeyKillCmd = func(containerName string) string { return workflow.WorkflowFullName + "/kill-cmd-" + containerName }

// GlobalVarWorkflowRootTags is a list of root tags in workflow which could be used for variable reference
var GlobalVarValidWorkflowVariablePrefix = []string{"item.", "steps.", "inputs.", "outputs.", "pod.", "workflow.", "tasks."}

func UnstructuredHasCompletedLabel(obj interface{}) bool {
	if wf, ok := obj.(*unstructured.Unstructured); ok {
		return wf.GetLabels()[LabelKeyCompleted] == "true"
	}
	return false
}
