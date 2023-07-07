# IoArgoprojWorkflowV1alpha1WorkflowSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active_deadline_seconds** | Option<**i32**> | Optional duration in seconds relative to the workflow start time which the workflow is allowed to run before the controller terminates the io.argoproj.workflow.v1alpha1. A value of zero is used to terminate a Running workflow | [optional]
**affinity** | Option<[**crate::models::Affinity**](Affinity.md)> |  | [optional]
**archive_logs** | Option<**bool**> | ArchiveLogs indicates if the container logs should be archived | [optional]
**arguments** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Arguments**](io.argoproj.workflow.v1alpha1.Arguments.md)> |  | [optional]
**artifact_gc** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1WorkflowLevelArtifactGc**](io.argoproj.workflow.v1alpha1.WorkflowLevelArtifactGC.md)> |  | [optional]
**artifact_repository_ref** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ArtifactRepositoryRef**](io.argoproj.workflow.v1alpha1.ArtifactRepositoryRef.md)> |  | [optional]
**automount_service_account_token** | Option<**bool**> | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false. | [optional]
**dns_config** | Option<[**crate::models::PodDnsConfig**](PodDNSConfig.md)> |  | [optional]
**dns_policy** | Option<**String**> | Set DNS policy for the pod. Defaults to \"ClusterFirst\". Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to 'ClusterFirstWithHostNet'. | [optional]
**entrypoint** | Option<**String**> | Entrypoint is a template reference to the starting point of the io.argoproj.workflow.v1alpha1. | [optional]
**executor** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ExecutorConfig**](io.argoproj.workflow.v1alpha1.ExecutorConfig.md)> |  | [optional]
**hooks** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojWorkflowV1alpha1LifecycleHook>**](io.argoproj.workflow.v1alpha1.LifecycleHook.md)> | Hooks holds the lifecycle hook which is invoked at lifecycle of step, irrespective of the success, failure, or error status of the primary step | [optional]
**host_aliases** | Option<[**Vec<crate::models::HostAlias>**](HostAlias.md)> |  | [optional]
**host_network** | Option<**bool**> | Host networking requested for this workflow pod. Default to false. | [optional]
**image_pull_secrets** | Option<[**Vec<crate::models::LocalObjectReference>**](LocalObjectReference.md)> | ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet. More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod | [optional]
**metrics** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Metrics**](io.argoproj.workflow.v1alpha1.Metrics.md)> |  | [optional]
**node_selector** | Option<**::std::collections::HashMap<String, String>**> | NodeSelector is a selector which will result in all pods of the workflow to be scheduled on the selected node(s). This is able to be overridden by a nodeSelector specified in the template. | [optional]
**on_exit** | Option<**String**> | OnExit is a template reference which is invoked at the end of the workflow, irrespective of the success, failure, or error of the primary io.argoproj.workflow.v1alpha1. | [optional]
**parallelism** | Option<**i32**> | Parallelism limits the max total parallel pods that can execute at the same time in a workflow | [optional]
**pod_disruption_budget** | Option<[**crate::models::IoK8sApiPolicyV1PodDisruptionBudgetSpec**](io.k8s.api.policy.v1.PodDisruptionBudgetSpec.md)> |  | [optional]
**pod_gc** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1PodGc**](io.argoproj.workflow.v1alpha1.PodGC.md)> |  | [optional]
**pod_metadata** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Metadata**](io.argoproj.workflow.v1alpha1.Metadata.md)> |  | [optional]
**pod_priority** | Option<**i32**> | Priority to apply to workflow pods. DEPRECATED: Use PodPriorityClassName instead. | [optional]
**pod_priority_class_name** | Option<**String**> | PriorityClassName to apply to workflow pods. | [optional]
**pod_spec_patch** | Option<**String**> | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits). | [optional]
**priority** | Option<**i32**> | Priority is used if controller is configured to process limited number of workflows in parallel. Workflows with higher priority are processed first. | [optional]
**retry_strategy** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1RetryStrategy**](io.argoproj.workflow.v1alpha1.RetryStrategy.md)> |  | [optional]
**scheduler_name** | Option<**String**> | Set scheduler name for all pods. Will be overridden if container/script template's scheduler name is set. Default scheduler will be used if neither specified. | [optional]
**security_context** | Option<[**crate::models::PodSecurityContext**](PodSecurityContext.md)> |  | [optional]
**service_account_name** | Option<**String**> | ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as. | [optional]
**shutdown** | Option<**String**> | Shutdown will shutdown the workflow according to its ShutdownStrategy | [optional]
**suspend** | Option<**bool**> | Suspend will suspend the workflow and prevent execution of any future steps in the workflow | [optional]
**synchronization** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Synchronization**](io.argoproj.workflow.v1alpha1.Synchronization.md)> |  | [optional]
**template_defaults** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Template**](io.argoproj.workflow.v1alpha1.Template.md)> |  | [optional]
**templates** | Option<[**Vec<crate::models::IoArgoprojWorkflowV1alpha1Template>**](io.argoproj.workflow.v1alpha1.Template.md)> | Templates is a list of workflow templates used in a workflow | [optional]
**tolerations** | Option<[**Vec<crate::models::Toleration>**](Toleration.md)> | Tolerations to apply to workflow pods. | [optional]
**ttl_strategy** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1TtlStrategy**](io.argoproj.workflow.v1alpha1.TTLStrategy.md)> |  | [optional]
**volume_claim_gc** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1VolumeClaimGc**](io.argoproj.workflow.v1alpha1.VolumeClaimGC.md)> |  | [optional]
**volume_claim_templates** | Option<[**Vec<crate::models::PersistentVolumeClaim>**](PersistentVolumeClaim.md)> | VolumeClaimTemplates is a list of claims that containers are allowed to reference. The Workflow controller will create the claims at the beginning of the workflow and delete the claims upon completion of the workflow | [optional]
**volumes** | Option<[**Vec<crate::models::Volume>**](Volume.md)> | Volumes is a list of volumes that can be mounted by containers in a io.argoproj.workflow.v1alpha1. | [optional]
**workflow_metadata** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1WorkflowMetadata**](io.argoproj.workflow.v1alpha1.WorkflowMetadata.md)> |  | [optional]
**workflow_template_ref** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1WorkflowTemplateRef**](io.argoproj.workflow.v1alpha1.WorkflowTemplateRef.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


