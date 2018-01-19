package common

import (
	"time"

	"github.com/argoproj/argo/pkg/apis/workflow"
)

const (
	// DefaultControllerDeploymentName is the default deployment name of the workflow controller
	DefaultControllerDeploymentName = "workflow-controller"
	// DefaultControllerNamespace is the default namespace where the workflow controller is installed
	DefaultControllerNamespace = "kube-system"

	// WorkflowControllerConfigMapKey is the key in the configmap to retrieve workflow configuration from.
	// Content encoding is expected to be YAML.
	WorkflowControllerConfigMapKey = "config"

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

	// DockerLibVolumeName is the volume name for the /var/lib/docker host path volume
	DockerLibVolumeName = "docker-lib"
	// DockerLibHostPath is the host directory path containing docker runtime state
	DockerLibHostPath = "/var/lib/docker"
	// DockerSockVolumeName is the volume name for the /var/run/docker.sock host path volume
	DockerSockVolumeName = "docker-sock"

	// AnnotationKeyNodeName is the pod metadata annotation key containing the workflow node name
	AnnotationKeyNodeName = workflow.FullName + "/node-name"
	// AnnotationKeyNodeMessage is the pod metadata annotation key the executor will use to
	// communicate errors encountered by the executor during artifact load/save, etc...
	AnnotationKeyNodeMessage = workflow.FullName + "/node-message"
	// AnnotationKeyTemplate is the pod metadata annotation key containing the container template as JSON
	AnnotationKeyTemplate = workflow.FullName + "/template"
	// AnnotationKeyOutputs is the pod metadata annotation key containing the container outputs
	AnnotationKeyOutputs = workflow.FullName + "/outputs"
	// AnnotationKeyExecutionControl is the pod metadata annotation key containing execution control parameters
	// set by the controller and obeyed by the executor. For example, the controller will use this annotation to
	// signal the executors of daemoned containers that it should terminate.
	AnnotationKeyExecutionControl = workflow.FullName + "/execution"

	// LabelKeyControllerInstanceID is the label the controller will carry forward to pod labels
	// for the purposes of workflow segregation
	LabelKeyControllerInstanceID = workflow.FullName + "/controller-instanceid"
	// LabelKeyCompleted is the metadata label applied on worfklows and workflow pods to indicates if resource is completed
	// Workflows and pods with a completed=true label will be ignored by the controller
	LabelKeyCompleted = workflow.FullName + "/completed"
	// LabelKeyWorkflow is the pod metadata label to indicate the associated workflow name
	LabelKeyWorkflow = workflow.FullName + "/workflow"
	// LabelKeyPhase is a label applied to workflows to indicate the current phase of the workflow (for filtering purposes)
	LabelKeyPhase = workflow.FullName + "/phase"

	// ExecutorArtifactBaseDir is the base directory in the init container in which artifacts will be copied to.
	// Each artifact will be named according to its input name (e.g: /argo/inputs/artifacts/CODE)
	ExecutorArtifactBaseDir = "/argo/inputs/artifacts"

	// InitContainerMainFilesystemDir is a path made available to the init container such that the init container
	// can access the same volume mounts used in the main container. This is used for the purposes of artifact loading
	// (when there is overlapping paths between artifacts and volume mounts)
	InitContainerMainFilesystemDir = "/mainctrfs"

	// ExecutorStagingEmptyDir is the path of the emptydir which is used as a staging area to transfer a file between init/main container for script/resource templates
	ExecutorStagingEmptyDir = "/argo/staging"
	// ExecutorScriptSourcePath is the path which init will write the script source file to for script templates
	ExecutorScriptSourcePath = "/argo/staging/script"
	// ExecutorResourceManifestPath is the path which init will write the a manifest file to for resource templates
	ExecutorResourceManifestPath = "/tmp/manifest.yaml"

	// Various environment variables containing pod information exposed to the executor container(s)

	// EnvVarPodIP contains the IP of the pod (currently unused)
	EnvVarPodIP = "ARGO_POD_IP"
	// EnvVarPodName contains the name of the pod (currently unused)
	EnvVarPodName = "ARGO_POD_NAME"
	// EnvVarNamespace contains the namespace of the pod (currently unused)
	EnvVarNamespace = "ARGO_NAMESPACE"

	// These are global variables that are added to the scope during template execution and can be referenced using {{}} syntax

	// GlobalVarWorkflowName is a global workflow variable referencing the workflow's metadata.name field
	GlobalVarWorkflowName = "workflow.name"
	// GlobalVarWorkflowUID is a global workflow variable referencing the workflow's metadata.uid field
	GlobalVarWorkflowUID = "workflow.uid"
	// GlobalVarWorkflowStatus is a global workflow variable referencing the workflow's status.phase field
	GlobalVarWorkflowStatus = "workflow.status"
)

// ExecutionControl contains execution control parameters for executor to decide how to execute the container
type ExecutionControl struct {
	// Deadline is a max timestamp in which an executor can run the container before terminating it
	// It is used to signal the executor to terminate a daemoned container. In the future it will be
	// used to support workflow or steps/dag level timeouts.
	Deadline *time.Time `json:"deadline,omitempty"`
}
