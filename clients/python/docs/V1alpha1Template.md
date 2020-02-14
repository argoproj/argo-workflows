# V1alpha1Template

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active_deadline_seconds** | **str** | Optional duration in seconds relative to the StartTime that the pod may be active on a node before the system actively tries to terminate the pod; value must be positive integer This field is only applicable to container and script templates. | [optional] 
**affinity** | [**V1Affinity**](V1Affinity.md) |  | [optional] 
**archive_location** | [**V1alpha1ArtifactLocation**](V1alpha1ArtifactLocation.md) |  | [optional] 
**arguments** | [**V1alpha1Arguments**](V1alpha1Arguments.md) |  | [optional] 
**automount_service_account_token** | **bool** | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false. | [optional] 
**container** | [**V1Container**](V1Container.md) |  | [optional] 
**daemon** | **bool** |  | [optional] 
**dag** | [**V1alpha1DAGTemplate**](V1alpha1DAGTemplate.md) |  | [optional] 
**executor** | [**V1alpha1ExecutorConfig**](V1alpha1ExecutorConfig.md) |  | [optional] 
**host_aliases** | [**list[V1HostAlias]**](V1HostAlias.md) |  | [optional] 
**init_containers** | [**list[V1alpha1UserContainer]**](V1alpha1UserContainer.md) |  | [optional] 
**inputs** | [**V1alpha1Inputs**](V1alpha1Inputs.md) |  | [optional] 
**metadata** | [**V1alpha1Metadata**](V1alpha1Metadata.md) |  | [optional] 
**name** | **str** |  | [optional] 
**node_selector** | **dict(str, str)** | NodeSelector is a selector to schedule this step of the workflow to be run on the selected node(s). Overrides the selector set at the workflow level. | [optional] 
**outputs** | [**V1alpha1Outputs**](V1alpha1Outputs.md) |  | [optional] 
**parallelism** | **str** | Parallelism limits the max total parallel pods that can execute at the same time within the boundaries of this template invocation. If additional steps/dag templates are invoked, the pods created by those templates will not be counted towards this total. | [optional] 
**pod_spec_patch** | **str** | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits). | [optional] 
**priority** | **int** | Priority to apply to workflow pods. | [optional] 
**priority_class_name** | **str** | PriorityClassName to apply to workflow pods. | [optional] 
**resource** | [**V1alpha1ResourceTemplate**](V1alpha1ResourceTemplate.md) |  | [optional] 
**retry_strategy** | [**V1alpha1RetryStrategy**](V1alpha1RetryStrategy.md) |  | [optional] 
**scheduler_name** | **str** |  | [optional] 
**script** | [**V1alpha1ScriptTemplate**](V1alpha1ScriptTemplate.md) |  | [optional] 
**security_context** | [**V1PodSecurityContext**](V1PodSecurityContext.md) |  | [optional] 
**service_account_name** | **str** |  | [optional] 
**sidecars** | [**list[V1alpha1UserContainer]**](V1alpha1UserContainer.md) |  | [optional] 
**steps** | [**list[V1alpha1ParallelSteps]**](V1alpha1ParallelSteps.md) |  | [optional] 
**suspend** | [**V1alpha1SuspendTemplate**](V1alpha1SuspendTemplate.md) |  | [optional] 
**template** | **str** | Template is the name of the template which is used as the base of this template. | [optional] 
**template_ref** | [**V1alpha1TemplateRef**](V1alpha1TemplateRef.md) |  | [optional] 
**tolerations** | [**list[V1Toleration]**](V1Toleration.md) |  | [optional] 
**volumes** | [**list[V1Volume]**](V1Volume.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


