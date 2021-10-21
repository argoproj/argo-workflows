# V1alpha1Template

Template is a reusable and composable unit of execution in a workflow
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | [**V1Affinity**](V1Affinity.md) |  | [optional] 
**archive_location** | [**V1alpha1ArtifactLocation**](V1alpha1ArtifactLocation.md) |  | [optional] 
**arguments** | [**V1alpha1Arguments**](V1alpha1Arguments.md) |  | [optional] 
**automount_service_account_token** | **bool** | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false. | [optional] 
**container** | [**V1Container**](V1Container.md) |  | [optional] 
**daemon** | **bool** | Deamon will allow a workflow to proceed to the next step so long as the container reaches readiness | [optional] 
**dag** | [**V1alpha1DAGTemplate**](V1alpha1DAGTemplate.md) |  | [optional] 
**executor** | [**V1alpha1ExecutorConfig**](V1alpha1ExecutorConfig.md) |  | [optional] 
**host_aliases** | [**list[V1HostAlias]**](V1HostAlias.md) | HostAliases is an optional list of hosts and IPs that will be injected into the pod spec | [optional] 
**init_containers** | [**list[V1alpha1UserContainer]**](V1alpha1UserContainer.md) | InitContainers is a list of containers which run before the main container. | [optional] 
**inputs** | [**V1alpha1Inputs**](V1alpha1Inputs.md) |  | [optional] 
**memoize** | [**V1alpha1Memoize**](V1alpha1Memoize.md) |  | [optional] 
**metadata** | [**V1alpha1Metadata**](V1alpha1Metadata.md) |  | [optional] 
**metrics** | [**V1alpha1Metrics**](V1alpha1Metrics.md) |  | [optional] 
**name** | **str** | Name is the name of the template | 
**node_selector** | **dict(str, str)** | NodeSelector is a selector to schedule this step of the workflow to be run on the selected node(s). Overrides the selector set at the workflow level. | [optional] 
**outputs** | [**V1alpha1Outputs**](V1alpha1Outputs.md) |  | [optional] 
**parallelism** | **int** | Parallelism limits the max total parallel pods that can execute at the same time within the boundaries of this template invocation. If additional steps/dag templates are invoked, the pods created by those templates will not be counted towards this total. | [optional] 
**pod_spec_patch** | **str** | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits). | [optional] 
**priority** | **int** | Priority to apply to workflow pods. | [optional] 
**priority_class_name** | **str** | PriorityClassName to apply to workflow pods. | [optional] 
**resource** | [**V1alpha1ResourceTemplate**](V1alpha1ResourceTemplate.md) |  | [optional] 
**retry_strategy** | [**V1alpha1RetryStrategy**](V1alpha1RetryStrategy.md) |  | [optional] 
**scheduler_name** | **str** | If specified, the pod will be dispatched by specified scheduler. Or it will be dispatched by workflow scope scheduler if specified. If neither specified, the pod will be dispatched by default scheduler. | [optional] 
**script** | [**V1alpha1ScriptTemplate**](V1alpha1ScriptTemplate.md) |  | [optional] 
**security_context** | [**V1PodSecurityContext**](V1PodSecurityContext.md) |  | [optional] 
**service_account_name** | **str** | ServiceAccountName to apply to workflow pods | [optional] 
**sidecars** | [**list[V1alpha1UserContainer]**](V1alpha1UserContainer.md) | Sidecars is a list of containers which run alongside the main container Sidecars are automatically killed when the main container completes | [optional] 
**steps** | **list[list[V1alpha1WorkflowStep]]** | Steps define a series of sequential/parallel workflow steps | [optional] 
**suspend** | [**V1alpha1SuspendTemplate**](V1alpha1SuspendTemplate.md) |  | [optional] 
**synchronization** | [**V1alpha1Synchronization**](V1alpha1Synchronization.md) |  | [optional] 
**template** | **str** | Template is the name of the template which is used as the base of this template. DEPRECATED: This field is not used. | [optional] 
**template_ref** | [**V1alpha1TemplateRef**](V1alpha1TemplateRef.md) |  | [optional] 
**timeout** | **str** | Timout allows to set the total node execution timeout duration counting from the node&#39;s start time. This duration also includes time in which the node spends in Pending state. This duration may not be applied to Step or DAG templates. | [optional] 
**tolerations** | [**list[V1Toleration]**](V1Toleration.md) | Tolerations to apply to workflow pods. | [optional] 
**volumes** | [**list[V1Volume]**](V1Volume.md) | Volumes is a list of volumes that can be mounted by containers in a template. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


