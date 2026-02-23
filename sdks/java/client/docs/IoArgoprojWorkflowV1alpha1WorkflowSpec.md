

# IoArgoprojWorkflowV1alpha1WorkflowSpec

WorkflowSpec is the specification of a Workflow.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**activeDeadlineSeconds** | **Integer** | Optional duration in seconds relative to the workflow start time which the workflow is allowed to run before the controller terminates the io.argoproj.workflow.v1alpha1. A value of zero is used to terminate a Running workflow |  [optional]
**affinity** | [**io.kubernetes.client.openapi.models.V1Affinity**](io.kubernetes.client.openapi.models.V1Affinity.md) |  |  [optional]
**archiveLogs** | **Boolean** | ArchiveLogs indicates if the container logs should be archived |  [optional]
**arguments** | [**IoArgoprojWorkflowV1alpha1Arguments**](IoArgoprojWorkflowV1alpha1Arguments.md) |  |  [optional]
**artifactGC** | [**IoArgoprojWorkflowV1alpha1WorkflowLevelArtifactGC**](IoArgoprojWorkflowV1alpha1WorkflowLevelArtifactGC.md) |  |  [optional]
**artifactRepositoryRef** | [**IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef**](IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef.md) |  |  [optional]
**automountServiceAccountToken** | **Boolean** | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false. |  [optional]
**dnsConfig** | [**io.kubernetes.client.openapi.models.V1PodDNSConfig**](io.kubernetes.client.openapi.models.V1PodDNSConfig.md) |  |  [optional]
**dnsPolicy** | **String** | Set DNS policy for workflow pods. Defaults to \&quot;ClusterFirst\&quot;. Valid values are &#39;ClusterFirstWithHostNet&#39;, &#39;ClusterFirst&#39;, &#39;Default&#39; or &#39;None&#39;. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to &#39;ClusterFirstWithHostNet&#39;. |  [optional]
**entrypoint** | **String** | Entrypoint is a template reference to the starting point of the io.argoproj.workflow.v1alpha1. |  [optional]
**executor** | [**IoArgoprojWorkflowV1alpha1ExecutorConfig**](IoArgoprojWorkflowV1alpha1ExecutorConfig.md) |  |  [optional]
**executorPlugins** | [**List&lt;IoArgoprojWorkflowV1alpha1ExecutorPlugin&gt;**](IoArgoprojWorkflowV1alpha1ExecutorPlugin.md) | Specifies executor plugins at the workflow level.  This field is effective only when the ARGO_WORKFLOW_LEVEL_EXECUTOR_PLUGINS feature gate is enabled.  If this field is present (even if empty), executor plugin settings from the controller ConfigMap are ignored.  If this field is not set, the controller falls back to the ConfigMap configuration. |  [optional]
**hooks** | [**Map&lt;String, IoArgoprojWorkflowV1alpha1LifecycleHook&gt;**](IoArgoprojWorkflowV1alpha1LifecycleHook.md) | Hooks holds the lifecycle hook which is invoked at lifecycle of step, irrespective of the success, failure, or error status of the primary step |  [optional]
**hostAliases** | [**List&lt;io.kubernetes.client.openapi.models.V1HostAlias&gt;**](io.kubernetes.client.openapi.models.V1HostAlias.md) |  |  [optional]
**hostNetwork** | **Boolean** | Host networking requested for this workflow pod. Default to false. |  [optional]
**imagePullSecrets** | [**List&lt;io.kubernetes.client.openapi.models.V1LocalObjectReference&gt;**](io.kubernetes.client.openapi.models.V1LocalObjectReference.md) | ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet. More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod |  [optional]
**metrics** | [**IoArgoprojWorkflowV1alpha1Metrics**](IoArgoprojWorkflowV1alpha1Metrics.md) |  |  [optional]
**nodeSelector** | **Map&lt;String, String&gt;** | NodeSelector is a selector which will result in all pods of the workflow to be scheduled on the selected node(s). This is able to be overridden by a nodeSelector specified in the template. |  [optional]
**onExit** | **String** | OnExit is a template reference which is invoked at the end of the workflow, irrespective of the success, failure, or error of the primary io.argoproj.workflow.v1alpha1. |  [optional]
**parallelism** | **Integer** | Parallelism limits the max total parallel pods that can execute at the same time in a workflow |  [optional]
**podDisruptionBudget** | [**IoK8sApiPolicyV1PodDisruptionBudgetSpec**](IoK8sApiPolicyV1PodDisruptionBudgetSpec.md) |  |  [optional]
**podGC** | [**IoArgoprojWorkflowV1alpha1PodGC**](IoArgoprojWorkflowV1alpha1PodGC.md) |  |  [optional]
**podMetadata** | [**IoArgoprojWorkflowV1alpha1Metadata**](IoArgoprojWorkflowV1alpha1Metadata.md) |  |  [optional]
**podPriorityClassName** | **String** | PriorityClassName to apply to workflow pods. |  [optional]
**podSpecPatch** | **String** | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits). |  [optional]
**priority** | **Integer** | Priority is used if controller is configured to process limited number of workflows in parallel. Workflows with higher priority are processed first. |  [optional]
**retryStrategy** | [**IoArgoprojWorkflowV1alpha1RetryStrategy**](IoArgoprojWorkflowV1alpha1RetryStrategy.md) |  |  [optional]
**schedulerName** | **String** | Set scheduler name for all pods. Will be overridden if container/script template&#39;s scheduler name is set. Default scheduler will be used if neither specified. |  [optional]
**securityContext** | [**io.kubernetes.client.openapi.models.V1PodSecurityContext**](io.kubernetes.client.openapi.models.V1PodSecurityContext.md) |  |  [optional]
**serviceAccountName** | **String** | ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as. |  [optional]
**shutdown** | **String** | Shutdown will shutdown the workflow according to its ShutdownStrategy |  [optional]
**suspend** | **Boolean** | Suspend will suspend the workflow and prevent execution of any future steps in the workflow |  [optional]
**synchronization** | [**IoArgoprojWorkflowV1alpha1Synchronization**](IoArgoprojWorkflowV1alpha1Synchronization.md) |  |  [optional]
**templateDefaults** | [**IoArgoprojWorkflowV1alpha1Template**](IoArgoprojWorkflowV1alpha1Template.md) |  |  [optional]
**templates** | [**List&lt;IoArgoprojWorkflowV1alpha1Template&gt;**](IoArgoprojWorkflowV1alpha1Template.md) | Templates is a list of workflow templates used in a workflow MaxItems is an artificial limit to limit CEL validation costs - see note at top of file |  [optional]
**tolerations** | [**List&lt;io.kubernetes.client.openapi.models.V1Toleration&gt;**](io.kubernetes.client.openapi.models.V1Toleration.md) | Tolerations to apply to workflow pods. |  [optional]
**ttlStrategy** | [**IoArgoprojWorkflowV1alpha1TTLStrategy**](IoArgoprojWorkflowV1alpha1TTLStrategy.md) |  |  [optional]
**volumeClaimGC** | [**IoArgoprojWorkflowV1alpha1VolumeClaimGC**](IoArgoprojWorkflowV1alpha1VolumeClaimGC.md) |  |  [optional]
**volumeClaimTemplates** | [**List&lt;io.kubernetes.client.openapi.models.V1PersistentVolumeClaim&gt;**](io.kubernetes.client.openapi.models.V1PersistentVolumeClaim.md) | VolumeClaimTemplates is a list of claims that containers are allowed to reference. The Workflow controller will create the claims at the beginning of the workflow and delete the claims upon completion of the workflow |  [optional]
**volumes** | [**List&lt;io.kubernetes.client.openapi.models.V1Volume&gt;**](io.kubernetes.client.openapi.models.V1Volume.md) | Volumes is a list of volumes that can be mounted by containers in a io.argoproj.workflow.v1alpha1. |  [optional]
**workflowMetadata** | [**IoArgoprojWorkflowV1alpha1WorkflowMetadata**](IoArgoprojWorkflowV1alpha1WorkflowMetadata.md) |  |  [optional]
**workflowTemplateRef** | [**IoArgoprojWorkflowV1alpha1WorkflowTemplateRef**](IoArgoprojWorkflowV1alpha1WorkflowTemplateRef.md) |  |  [optional]



