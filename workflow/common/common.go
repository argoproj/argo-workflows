package common

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/argoproj/argo/pkg/apis/workflow"
)

const (
	// DefaultArchivePattern is the default pattern when storing artifacts in an archive repository
	DefaultArchivePattern = "{{workflow.name}}/{{pod.name}}"

	// Container names used in the workflow pod
	MainContainerName = "main"
	InitContainerName = "init"
	WaitContainerName = "wait"

	// PodMetadataVolumeName is the volume name defined in a workflow pod spec to expose pod metadata via downward API
	PodMetadataVolumeName = "podmetadata"

	// PodMetadataAnnotationsVolumePath is volume path for metadata.annotations in the downward API
	PodMetadataAnnotationsVolumePath = "annotations"
	// PodMetadataMountPath is the directory mount location for DownwardAPI volume containing pod metadata
	PodMetadataMountPath = "/argo/" + PodMetadataVolumeName
	// PodMetadataAnnotationsPath is the file path containing pod metadata annotations. Examined by executor
	PodMetadataAnnotationsPath = PodMetadataMountPath + "/" + PodMetadataAnnotationsVolumePath

	// DockerSockVolumeName is the volume name for the /var/run/docker.sock host path volume
	DockerSockVolumeName = "docker-sock"

	// AnnotationKeyNodeName is the pod metadata annotation key containing the workflow node name
	AnnotationKeyNodeName = workflow.WorkflowFullName + "/node-name"
	// AnnotationKeyNodeName is the node's type
	AnnotationKeyNodeType = workflow.WorkflowFullName + "/node-type"

	// AnnotationKeyNodeMessage is the pod metadata annotation key the executor will use to
	// communicate errors encountered by the executor during artifact load/save, etc...
	AnnotationKeyNodeMessage = workflow.WorkflowFullName + "/node-message"
	// AnnotationKeyTemplate is the pod metadata annotation key containing the container template as JSON
	AnnotationKeyTemplate = workflow.WorkflowFullName + "/template"
	// AnnotationKeyOutputs is the pod metadata annotation key containing the container outputs
	AnnotationKeyOutputs = workflow.WorkflowFullName + "/outputs"
	// AnnotationKeyExecutionControl is the pod metadata annotation key containing execution control parameters
	// set by the controller and obeyed by the executor. For example, the controller will use this annotation to
	// signal the executors of daemoned containers that it should terminate.
	AnnotationKeyExecutionControl = workflow.WorkflowFullName + "/execution"

	// LabelKeyControllerInstanceID is the label the controller will carry forward to workflows/pod labels
	// for the purposes of workflow segregation
	LabelKeyControllerInstanceID = workflow.WorkflowFullName + "/controller-instanceid"
	// Who created this workflow.
	LabelKeyCreator = workflow.WorkflowFullName + "/creator"
	// LabelKeyCompleted is the metadata label applied on worfklows and workflow pods to indicates if resource is completed
	// Workflows and pods with a completed=true label will be ignored by the controller
	LabelKeyCompleted = workflow.WorkflowFullName + "/completed"
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

	KubeConfigDefaultMountPath    = "/kube/config"
	KubeConfigDefaultVolumeName   = "kubeconfig"
	ServiceAccountTokenMountPath  = "/var/run/secrets/kubernetes.io/serviceaccount"
	ServiceAccountTokenVolumeName = "exec-sa-token"
	SecretVolMountPath            = "/argo/secret"
)

// GlobalVarWorkflowRootTags is a list of root tags in workflow which could be used for variable reference
var GlobalVarValidWorkflowVariablePrefix = []string{"item.", "steps.", "inputs.", "outputs.", "pod.", "workflow.", "tasks."}

// ExecutionControl contains execution control parameters for executor to decide how to execute the container
type ExecutionControl struct {
	// Deadline is a max timestamp in which an executor can run the container before terminating it
	// It is used to signal the executor to terminate a daemoned container. In the future it will be
	// used to support workflow or steps/dag level timeouts.
	Deadline *time.Time `json:"deadline,omitempty"`
	// IncludeScriptOutput is containing flag to include script output
	IncludeScriptOutput bool `json:"includeScriptOutput,omitempty"`
}

func UnstructuredHasCompletedLabel(obj interface{}) bool {
	if wf, ok := obj.(*unstructured.Unstructured); ok {
		return wf.GetLabels()[LabelKeyCompleted] == "true"
	}
	return false
}
