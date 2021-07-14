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

	// DockerSockVolumeName is the volume name for the /var/run/docker.sock host path volume
	DockerSockVolumeName = "docker-sock"

	// AnnotationKeyNodeName is the pod metadata annotation key containing the workflow node name
	AnnotationKeyNodeName = workflow.WorkflowFullName + "/node-name"
	// AnnotationKeyNodeName is the node's type
	AnnotationKeyNodeType = workflow.WorkflowFullName + "/node-type"

	// AnnotationKeyRBACRule is a rule to match the claims
	AnnotationKeyRBACRule           = workflow.WorkflowFullName + "/rbac-rule"
	AnnotationKeyRBACRulePrecedence = workflow.WorkflowFullName + "/rbac-rule-precedence"

	// AnnotationKeyOutputs is the pod metadata annotation key containing the container outputs
	AnnotationKeyOutputs = workflow.WorkflowFullName + "/outputs"
	// AnnotationKeyCronWfScheduledTime is the workflow metadata annotation key containing the time when the workflow
	// was scheduled to run by CronWorkflow.
	AnnotationKeyCronWfScheduledTime = workflow.WorkflowFullName + "/scheduled-time"

	// LabelKeyControllerInstanceID is the label the controller will carry forward to workflows/pod labels
	// for the purposes of workflow segregation
	LabelKeyControllerInstanceID = workflow.WorkflowFullName + "/controller-instanceid"
	// Who created this workflow.
	LabelKeyCreator      = workflow.WorkflowFullName + "/creator"
	LabelKeyCreatorEmail = workflow.WorkflowFullName + "/creator-email"
	// LabelKeyCompleted is the metadata label applied on workflows and workflow pods to indicates if resource is completed
	// Workflows and pods with a completed=true label will be ignored by the controller.
	// See also `LabelKeyWorkflowArchivingStatus`.
	LabelKeyCompleted = workflow.WorkflowFullName + "/completed"
	// LabelKeyWorkflowArchivingStatus indicates if a workflow needs archiving or not:
	// * `` - does not need archiving ... yet
	// * `Pending` - pending archiving
	// * `Archived` - has been archived
	// See also `LabelKeyCompleted`.
	LabelKeyWorkflowArchivingStatus = workflow.WorkflowFullName + "/workflow-archiving-status"
	// LabelKeyWorkflow is the pod metadata label to indicate the associated workflow name
	LabelKeyWorkflow = workflow.WorkflowFullName + "/workflow"
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
	// LabelKeyWorkflowTemplate is a label applied to Workflows that are submitted from ClusterWorkflowtemplate
	LabelKeyClusterWorkflowTemplate = workflow.WorkflowFullName + "/cluster-workflow-template"
	// LabelKeyOnExit is a label applied to Pods that are run from onExit nodes, so that they are not shut down when stopping a Workflow
	LabelKeyOnExit = workflow.WorkflowFullName + "/on-exit"

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
	// ExecutorResourceManifestPath is the path which init will write the a manifest file to for resource templates
	ExecutorResourceManifestPath = "/tmp/manifest.yaml"

	// Various environment variables containing pod information exposed to the executor container(s)

	// EnvVarPodName contains the name of the pod (currently unused)
	EnvVarPodName = "ARGO_POD_NAME"
	// EnvVarContainerName container the container's name for the current pod
	EnvVarContainerName = "ARGO_CONTAINER_NAME"
	// EnvVarDeadline is the deadline for the pod
	EnvVarDeadline = "ARGO_DEADLINE"
	// EnvVarIncludeScriptOutput capture the stdout and stderr
	EnvVarIncludeScriptOutput = "ARGO_INCLUDE_SCRIPT_OUTPUT"
	// EnvVarTemplate is the template
	EnvVarTemplate = "ARGO_TEMPLATE"
	// EnvVarContainerRuntimeExecutor contains the name of the container runtime executor to use, empty is equal to "docker"
	EnvVarContainerRuntimeExecutor = "ARGO_CONTAINER_RUNTIME_EXECUTOR"
	// EnvVarDownwardAPINodeIP is the envvar used to get the `status.hostIP`
	EnvVarDownwardAPINodeIP = "ARGO_KUBELET_HOST"
	// EnvVarKubeletPort is used to configure the kubelet api port
	EnvVarKubeletPort = "ARGO_KUBELET_PORT"
	// EnvVarKubeletInsecure is used to disable the TLS verification
	EnvVarKubeletInsecure = "ARGO_KUBELET_INSECURE"
	// EnvVarArgoTrace is used enable tracing statements in Argo components
	EnvVarArgoTrace = "ARGO_TRACE"

	// ContainerRuntimeExecutorDocker to use docker as container runtime executor
	ContainerRuntimeExecutorDocker = "docker"

	// ContainerRuntimeExecutorKubelet to use the kubelet as container runtime executor
	ContainerRuntimeExecutorKubelet = "kubelet"

	// ContainerRuntimeExecutorK8sAPI to use the Kubernetes API server as container runtime executor
	ContainerRuntimeExecutorK8sAPI = "k8sapi"

	// ContainerRuntimeExecutorPNS indicates to use process namespace sharing as the container runtime executor
	ContainerRuntimeExecutorPNS = "pns"

	// ContainerRuntimeExecutorEmissary indicates to use emissary container runtime executor
	ContainerRuntimeExecutorEmissary = "emissary"

	// Variables that are added to the scope during template execution and can be referenced using {{}} syntax

	// GlobalVarWorkflowName is a global workflow variable referencing the workflow's metadata.name field
	GlobalVarWorkflowName = "workflow.name"
	// GlobalVarWorkflowNamespace is a global workflow variable referencing the workflow's metadata.namespace field
	GlobalVarWorkflowNamespace = "workflow.namespace"
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
	// GlobalVarWorkflowParameters is a JSON string containing all workflow parameters
	GlobalVarWorkflowParameters = "workflow.parameters"
	// GlobalVarWorkflowCronScheduleTime is the scheduled timestamp of a Workflow started by a CronWorkflow
	GlobalVarWorkflowCronScheduleTime = "workflow.scheduledTime"

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
	// LocalVarRetriesLastDuration is a variable that references information about the last retry's duration
	LocalVarRetriesLastDuration = "lastRetry.duration"

	KubeConfigDefaultMountPath    = "/kube/config"
	KubeConfigDefaultVolumeName   = "kubeconfig"
	ServiceAccountTokenMountPath  = "/var/run/secrets/kubernetes.io/serviceaccount"
	ServiceAccountTokenVolumeName = "exec-sa-token"
	SecretVolMountPath            = "/argo/secret"
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
