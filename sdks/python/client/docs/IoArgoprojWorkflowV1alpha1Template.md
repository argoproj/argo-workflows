# IoArgoprojWorkflowV1alpha1Template

Template is a reusable and composable unit of execution in a workflow

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active_deadline_seconds** | **str** |  | [optional] 
**affinity** | [**Affinity**](Affinity.md) |  | [optional] 
**annotations** | **{str: (str,)}** | Annotations is a list of annotations to add to the template at runtime | [optional] 
**archive_location** | [**IoArgoprojWorkflowV1alpha1ArtifactLocation**](IoArgoprojWorkflowV1alpha1ArtifactLocation.md) |  | [optional] 
**automount_service_account_token** | **bool** | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false. | [optional] 
**container** | [**Container**](Container.md) |  | [optional] 
**container_set** | [**IoArgoprojWorkflowV1alpha1ContainerSetTemplate**](IoArgoprojWorkflowV1alpha1ContainerSetTemplate.md) |  | [optional] 
**daemon** | **bool** | Daemon will allow a workflow to proceed to the next step so long as the container reaches readiness | [optional] 
**dag** | [**IoArgoprojWorkflowV1alpha1DAGTemplate**](IoArgoprojWorkflowV1alpha1DAGTemplate.md) |  | [optional] 
**data** | [**IoArgoprojWorkflowV1alpha1Data**](IoArgoprojWorkflowV1alpha1Data.md) |  | [optional] 
**executor** | [**IoArgoprojWorkflowV1alpha1ExecutorConfig**](IoArgoprojWorkflowV1alpha1ExecutorConfig.md) |  | [optional] 
**fail_fast** | **bool** | FailFast, if specified, will fail this template if any of its child pods has failed. This is useful for when this template is expanded with &#x60;withItems&#x60;, etc. | [optional] 
**host_aliases** | [**[HostAlias]**](HostAlias.md) | HostAliases is an optional list of hosts and IPs that will be injected into the pod spec | [optional] 
**http** | [**IoArgoprojWorkflowV1alpha1HTTP**](IoArgoprojWorkflowV1alpha1HTTP.md) |  | [optional] 
**init_containers** | [**[IoArgoprojWorkflowV1alpha1UserContainer]**](IoArgoprojWorkflowV1alpha1UserContainer.md) | InitContainers is a list of containers which run before the main container. | [optional] 
**inputs** | [**IoArgoprojWorkflowV1alpha1Inputs**](IoArgoprojWorkflowV1alpha1Inputs.md) |  | [optional] 
**memoize** | [**IoArgoprojWorkflowV1alpha1Memoize**](IoArgoprojWorkflowV1alpha1Memoize.md) |  | [optional] 
**metadata** | [**IoArgoprojWorkflowV1alpha1Metadata**](IoArgoprojWorkflowV1alpha1Metadata.md) |  | [optional] 
**metrics** | [**IoArgoprojWorkflowV1alpha1Metrics**](IoArgoprojWorkflowV1alpha1Metrics.md) |  | [optional] 
**name** | **str** | Name is the name of the template | [optional] 
**node_selector** | **{str: (str,)}** | NodeSelector is a selector to schedule this step of the workflow to be run on the selected node(s). Overrides the selector set at the workflow level. | [optional] 
**outputs** | [**IoArgoprojWorkflowV1alpha1Outputs**](IoArgoprojWorkflowV1alpha1Outputs.md) |  | [optional] 
**parallelism** | **int** | Parallelism limits the max total parallel pods that can execute at the same time within the boundaries of this template invocation. If additional steps/dag templates are invoked, the pods created by those templates will not be counted towards this total. | [optional] 
**plugin** | **bool, date, datetime, dict, float, int, list, str, none_type** | Plugin is an Object with exactly one key | [optional] 
**pod_metadata_patch** | **str** | PodMetadataPatch holds strategic merge patch to apply against the pod metadata. Allows parameterization of metadata fields Labels and Annotations, which are not strings. | [optional] 
**pod_spec_patch** | **str** | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits). | [optional] 
**priority_class_name** | **str** | PriorityClassName to apply to workflow pods. | [optional] 
**resource** | [**IoArgoprojWorkflowV1alpha1ResourceTemplate**](IoArgoprojWorkflowV1alpha1ResourceTemplate.md) |  | [optional] 
**retry_strategy** | [**IoArgoprojWorkflowV1alpha1RetryStrategy**](IoArgoprojWorkflowV1alpha1RetryStrategy.md) |  | [optional] 
**scheduler_name** | **str** | If specified, the pod will be dispatched by specified scheduler. Or it will be dispatched by workflow scope scheduler if specified. If neither specified, the pod will be dispatched by default scheduler. | [optional] 
**script** | [**IoArgoprojWorkflowV1alpha1ScriptTemplate**](IoArgoprojWorkflowV1alpha1ScriptTemplate.md) |  | [optional] 
**security_context** | [**PodSecurityContext**](PodSecurityContext.md) |  | [optional] 
**service_account_name** | **str** | ServiceAccountName to apply to workflow pods | [optional] 
**sidecars** | [**[IoArgoprojWorkflowV1alpha1UserContainer]**](IoArgoprojWorkflowV1alpha1UserContainer.md) | Sidecars is a list of containers which run alongside the main container Sidecars are automatically killed when the main container completes | [optional] 
**steps** | [**[IoArgoprojWorkflowV1alpha1ParallelSteps]**](IoArgoprojWorkflowV1alpha1ParallelSteps.md) | Steps define a series of sequential/parallel workflow steps | [optional] 
**suspend** | [**IoArgoprojWorkflowV1alpha1SuspendTemplate**](IoArgoprojWorkflowV1alpha1SuspendTemplate.md) |  | [optional] 
**synchronization** | [**IoArgoprojWorkflowV1alpha1Synchronization**](IoArgoprojWorkflowV1alpha1Synchronization.md) |  | [optional] 
**timeout** | **str** | Timeout allows to set the total node execution timeout duration counting from the node&#39;s start time. This duration also includes time in which the node spends in Pending state. This duration may not be applied to Step or DAG templates. | [optional] 
**tolerations** | [**[Toleration]**](Toleration.md) | Tolerations to apply to workflow pods. | [optional] 
**volumes** | [**[Volume]**](Volume.md) | Volumes is a list of volumes that can be mounted by containers in a template. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


