package common

import (
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
)

const (
	// Container names used in the workflow pod
	MainContainerName = "main"
	InitContainerName = "init"
	WaitContainerName = "wait"
	// SupervisorContainerName is the name of the init-less auxiliary container
	// that replaces `wait` when initlessPod is enabled.
	SupervisorContainerName     = "supervisor"
	ArtifactPluginSidecarPrefix = "artifact-plugin-"
	ArtifactPluginInitPrefix    = InitContainerName + "-artifact-"

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

	// LabelParallelismLimit is a label applied on namespace objects to control the per namespace parallelism.
	LabelParallelismLimit = workflow.WorkflowFullName + "/parallelism-limit"

	// AnnotationKeyPodGCStrategy is listed as an annotation on the Pod
	// the strategy for the pod, in case the pod is orphaned from its workflow
	AnnotationKeyPodGCStrategy = workflow.WorkflowFullName + "/pod-gc-strategy"

	// AnnotationKeyTraceID is added as an annotation to workflows and pods for the topmost telemetry trace-id
	AnnotationKeyTraceID = workflow.WorkflowFullName + "/trace-id"
	// AnnotationKeySpanID is added as an annotation to workflows and pods for their span-id
	AnnotationKeySpanID = workflow.WorkflowFullName + "/span-id"

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
	// EnvVarArtifactPluginNames is a comma-separated list of artifact plugin
	// container names. It is set on the artifact GC pod, on the legacy `wait`
	// container, and on the init-less `supervisor` container, which use it to
	// locate and (for wait/supervisor) drive Save on each plugin sidecar.
	EnvVarArtifactPluginNames = "ARGO_ARTIFACT_PLUGIN_NAMES"
	// EnvVarInputArtifactPluginNames is the comma-separated list of input artifact plugin names
	// the supervisor container must invoke for Load in init-less pod mode.
	EnvVarInputArtifactPluginNames = "ARGO_INPUT_ARTIFACT_PLUGIN_NAMES"
	// EnvVarWaitForReady tells the emissary to block on the /var/run/argo/status
	// marker before reading the template and exec'ing the user command.
	// Set by the controller on main-level containers in init-less pod mode.
	EnvVarWaitForReady = "ARGO_WAIT_FOR_READY"
	// EnvVarInitlessPod signals that the executor is running inside an
	// init-less pod (i.e. as `argoexec supervisor`). The controller sets it
	// on the supervisor container. Used to gate init-less-specific code
	// paths in the executor (e.g. the input-artifacts overlap fallback in
	// stageArchiveFile).
	EnvVarInitlessPod = "ARGO_INITLESS_POD"
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
	// EnvVarS3UploadThreads is the number of threads for artifact upload through S3. Default: 4.
	EnvVarS3UploadThreads = "ARTIFACT_S3_UPLOAD_THREADS"
	// EnvVarS3UploadPartSizeMiB is the size in MiB of the part of a MultiPartUpload. Default: let Minio calculate automatically (16MiB for file <= 156GiB).
	EnvVarS3UploadPartSizeMiB = "ARTIFACT_S3_UPLOAD_PART_SIZE_MIB"
	// EnvAgentTaskWorkers is the number of task workers for the agent pod
	EnvAgentTaskWorkers = "ARGO_AGENT_TASK_WORKERS"
	// EnvAgentPatchRate is the rate that the Argo Agent will patch the Workflow TaskSet
	EnvAgentPatchRate = "ARGO_AGENT_PATCH_RATE"

	// Finalizer to block deletion of the workflow if deletion of artifacts fail for some reason.
	FinalizerArtifactGC = workflow.WorkflowFullName + "/artifact-gc"

	// Finalizer blocks the deletion of pods until the controller captures their status.
	FinalizerPodStatus = workflow.WorkflowFullName + "/status"

	// LabelKeyConfigMapType is the label key for the type of configmap.
	LabelKeyConfigMapType = "workflows.argoproj.io/configmap-type"
	// LabelValueTypeConfigMapCache is a key for configmaps that are memoization cache.
	LabelValueTypeConfigMapCache = "Cache"
	// LabelValueTypeConfigMapParameter is a key for configmaps that contains parameter values.
	LabelValueTypeConfigMapParameter = "Parameter"
	// LabelValueTypeConfigMapExecutorPlugin is a key for configmaps that contains an executor plugin.
	LabelValueTypeConfigMapExecutorPlugin = "ExecutorPlugin"

	KubeConfigDefaultMountPath    = "/kube/config"
	KubeConfigDefaultVolumeName   = "kubeconfig"
	ServiceAccountTokenMountPath  = "/var/run/secrets/kubernetes.io/serviceaccount"
	ServiceAccountTokenVolumeName = "exec-sa-token"
	SecretVolMountPath            = "/argo/secret"
	EnvConfigMountPath            = "/argo/config"
	EnvVarTemplateOffloaded       = "offloaded"
	// EnvVarContainerArgsFile is set when container args are offloaded to a file
	EnvVarContainerArgsFile = "ARGO_CONTAINER_ARGS_FILE"

	// MaxEnvVarLen is the maximum size in bytes for environment variables and arguments
	// before they are offloaded to a ConfigMap or file. This limit is based on
	// Kubernetes' etcd value size limit and Linux exec() argument size limits.
	MaxEnvVarLen = 131072 // 128KB

	// CACertificatesVolumeMountName is the name of the secret that contains the CA certificates.
	CACertificatesVolumeMountName = "argo-workflows-agent-ca-certificates"

	// VarRunArgoPath is the standard path for the shared volume
	VarRunArgoPath = "/var/run/argo"

	// LegacyArgoExecBinPath is where the init container copies argoexec in the
	// legacy pod layout (it lives in the shared VarRunArgoPath emptyDir).
	LegacyArgoExecBinPath = VarRunArgoPath + "/argoexec"

	// ArgoExecBinImageVolumeName / ArgoExecBinMountPath / ArgoExecBinPath describe
	// the image volume that delivers the argoexec binary to main-level containers
	// in init-less pod mode (there is no init container to copy it into
	// VarRunArgoPath). argoexec lives at /bin/argoexec inside the argoexec image.
	ArgoExecBinImageVolumeName = "argoexec-bin"
	ArgoExecBinMountPath       = "/argo-bin"
	ArgoExecBinPath            = ArgoExecBinMountPath + "/bin/argoexec"

	// ArgoProgressPath defines the path to a file used for self reporting progress
	ArgoProgressPath = VarRunArgoPath + "/progress"

	// StatusMarkerPath is the marker file the supervisor writes atomically once
	// pre-main setup concludes (init-less pod mode). The emissary in main waits
	// for it before exec'ing the user command. Its contents encode the outcome:
	// empty means success; a non-empty body is the failure reason, which the
	// emissary logs before exiting non-zero.
	StatusMarkerPath = VarRunArgoPath + "/status"

	// ExitCodeSupervisorPreMainFailure is the exit code main's emissary uses
	// when the supervisor's status marker reports a failure before exec'ing the user
	// command (init-less pod mode). 65 = sysexits.h EX_DATAERR, chosen to be
	// distinct from the user command's likely codes (0-2, 126-128, 137, 143) so
	// the controller can attribute the failure to supervisor pre-main setup
	// rather than the user command. The controller (inferFailedReason) keys off
	// this value to surface the supervisor's real error instead of the
	// placeholder code on main.
	ExitCodeSupervisorPreMainFailure = 65

	ConfigMapName = "workflow-controller-configmap"

	// AbsentOptionalArgumentValue marks an argument whose value was a single pure reference to a
	// skipped/omitted node's output with no producer valueFrom.default (an absent optional). The
	// controller's reference resolution writes it (wfScope.markAbsentOptionalArgs) so that textual
	// substitution succeeds where the raw absent value would be a terminal error; ProcessArgs
	// interprets it at consumption time — when the consumed template is fully resolved — as
	// "unsupplied", letting the input's own default apply and failing terminally when there is
	// none. It is an internal controller contract: user-supplied values are never expected to
	// collide with it, and a collision merely makes the argument behave as if it were omitted.
	AbsentOptionalArgumentValue = "__argo-internal.absent-optional-output__"
)

// AnnotationKeyKillCmd specifies the command to use to kill to container, useful for injected sidecars
var AnnotationKeyKillCmd = func(containerName string) string { return workflow.WorkflowFullName + "/kill-cmd-" + containerName }

// GlobalVarValidWorkflowVariablePrefix is a list of root tags in workflow which could be used for variable reference.
var GlobalVarValidWorkflowVariablePrefix = []string{"item.", "steps.", "inputs.", "outputs.", "pod.", "workflow.", "tasks."}

func UnstructuredHasCompletedLabel(obj any) bool {
	if wf, ok := obj.(*unstructured.Unstructured); ok {
		return wf.GetLabels()[LabelKeyCompleted] == "true"
	}
	return false
}

func IsArtifactPluginSidecar(containerName string) bool {
	return strings.HasPrefix(containerName, ArtifactPluginSidecarPrefix)
}

func IsArgoAuxilliary(containerName string) bool {
	return containerName == WaitContainerName || containerName == SupervisorContainerName
}

func IsArgoSidecar(containerName string) bool {
	return IsArgoAuxilliary(containerName) || IsArtifactPluginSidecar(containerName)
}

func IsArtifactPluginInit(containerName string) bool {
	return strings.HasPrefix(containerName, ArtifactPluginInitPrefix)
}

// IsInitlessPod reports whether the executor is running under the init-less pod
// layout, signalled by the controller via EnvVarInitlessPod on the supervisor
// and main-level containers. Several output-staging and lifecycle code paths
// diverge from the legacy init-container + wait layout in this mode.
func IsInitlessPod() bool {
	return os.Getenv(EnvVarInitlessPod) == "true"
}

// JoinPluginNames serializes artifact-plugin sidecar names into the
// comma-separated form the controller writes to EnvVarArtifactPluginNames /
// EnvVarInputArtifactPluginNames. Paired with SplitPluginNames.
func JoinPluginNames(names []string) string {
	return strings.Join(names, ",")
}

// SplitPluginNames parses the comma-separated value written by JoinPluginNames,
// trimming surrounding whitespace and dropping empty entries so callers never
// see a blank name from a trailing or doubled comma.
func SplitPluginNames(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			names = append(names, p)
		}
	}
	return names
}

// ResolveTemplateEnvValue takes a raw ARGO_TEMPLATE env value and returns
// the actual template JSON. If raw is the offloaded sentinel, reads from
// the configmap mount at offloadDir/EnvVarTemplate; otherwise returns the
// raw value unchanged. Shared between the legacy init container
// (cmd/argoexec/executor) and the emissary (cmd/argoexec/commands) so the
// offload protocol stays in one place.
func ResolveTemplateEnvValue(raw string, offloadDir string) ([]byte, error) {
	if raw == EnvVarTemplateOffloaded {
		return os.ReadFile(filepath.Join(offloadDir, EnvVarTemplate))
	}
	return []byte(raw), nil
}
