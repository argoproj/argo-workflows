# IoArgoprojWorkflowV1alpha1Template

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active_deadline_seconds** | Option<**String**> |  | [optional]
**affinity** | Option<[**crate::models::Affinity**](Affinity.md)> |  | [optional]
**archive_location** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ArtifactLocation**](io.argoproj.workflow.v1alpha1.ArtifactLocation.md)> |  | [optional]
**automount_service_account_token** | Option<**bool**> | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false. | [optional]
**container** | Option<[**crate::models::Container**](Container.md)> |  | [optional]
**container_set** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ContainerSetTemplate**](io.argoproj.workflow.v1alpha1.ContainerSetTemplate.md)> |  | [optional]
**daemon** | Option<**bool**> | Daemon will allow a workflow to proceed to the next step so long as the container reaches readiness | [optional]
**dag** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1DagTemplate**](io.argoproj.workflow.v1alpha1.DAGTemplate.md)> |  | [optional]
**data** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Data**](io.argoproj.workflow.v1alpha1.Data.md)> |  | [optional]
**executor** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ExecutorConfig**](io.argoproj.workflow.v1alpha1.ExecutorConfig.md)> |  | [optional]
**fail_fast** | Option<**bool**> | FailFast, if specified, will fail this template if any of its child pods has failed. This is useful for when this template is expanded with `withItems`, etc. | [optional]
**host_aliases** | Option<[**Vec<crate::models::HostAlias>**](HostAlias.md)> | HostAliases is an optional list of hosts and IPs that will be injected into the pod spec | [optional]
**http** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Http**](io.argoproj.workflow.v1alpha1.HTTP.md)> |  | [optional]
**init_containers** | Option<[**Vec<crate::models::IoArgoprojWorkflowV1alpha1UserContainer>**](io.argoproj.workflow.v1alpha1.UserContainer.md)> | InitContainers is a list of containers which run before the main container. | [optional]
**inputs** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Inputs**](io.argoproj.workflow.v1alpha1.Inputs.md)> |  | [optional]
**memoize** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Memoize**](io.argoproj.workflow.v1alpha1.Memoize.md)> |  | [optional]
**metadata** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Metadata**](io.argoproj.workflow.v1alpha1.Metadata.md)> |  | [optional]
**metrics** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Metrics**](io.argoproj.workflow.v1alpha1.Metrics.md)> |  | [optional]
**name** | Option<**String**> | Name is the name of the template | [optional]
**node_selector** | Option<**::std::collections::HashMap<String, String>**> | NodeSelector is a selector to schedule this step of the workflow to be run on the selected node(s). Overrides the selector set at the workflow level. | [optional]
**outputs** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Outputs**](io.argoproj.workflow.v1alpha1.Outputs.md)> |  | [optional]
**parallelism** | Option<**i32**> | Parallelism limits the max total parallel pods that can execute at the same time within the boundaries of this template invocation. If additional steps/dag templates are invoked, the pods created by those templates will not be counted towards this total. | [optional]
**plugin** | Option<[**serde_json::Value**](.md)> | Plugin is an Object with exactly one key | [optional]
**pod_spec_patch** | Option<**String**> | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits). | [optional]
**priority** | Option<**i32**> | Priority to apply to workflow pods. | [optional]
**priority_class_name** | Option<**String**> | PriorityClassName to apply to workflow pods. | [optional]
**resource** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ResourceTemplate**](io.argoproj.workflow.v1alpha1.ResourceTemplate.md)> |  | [optional]
**retry_strategy** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1RetryStrategy**](io.argoproj.workflow.v1alpha1.RetryStrategy.md)> |  | [optional]
**scheduler_name** | Option<**String**> | If specified, the pod will be dispatched by specified scheduler. Or it will be dispatched by workflow scope scheduler if specified. If neither specified, the pod will be dispatched by default scheduler. | [optional]
**script** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1ScriptTemplate**](io.argoproj.workflow.v1alpha1.ScriptTemplate.md)> |  | [optional]
**security_context** | Option<[**crate::models::PodSecurityContext**](PodSecurityContext.md)> |  | [optional]
**service_account_name** | Option<**String**> | ServiceAccountName to apply to workflow pods | [optional]
**sidecars** | Option<[**Vec<crate::models::IoArgoprojWorkflowV1alpha1UserContainer>**](io.argoproj.workflow.v1alpha1.UserContainer.md)> | Sidecars is a list of containers which run alongside the main container Sidecars are automatically killed when the main container completes | [optional]
**steps** | Option<[**Vec<crate::models::IoArgoprojWorkflowV1alpha1ParallelSteps>**](io.argoproj.workflow.v1alpha1.ParallelSteps.md)> | Steps define a series of sequential/parallel workflow steps | [optional]
**suspend** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1SuspendTemplate**](io.argoproj.workflow.v1alpha1.SuspendTemplate.md)> |  | [optional]
**synchronization** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Synchronization**](io.argoproj.workflow.v1alpha1.Synchronization.md)> |  | [optional]
**timeout** | Option<**String**> | Timeout allows to set the total node execution timeout duration counting from the node's start time. This duration also includes time in which the node spends in Pending state. This duration may not be applied to Step or DAG templates. | [optional]
**tolerations** | Option<[**Vec<crate::models::Toleration>**](Toleration.md)> | Tolerations to apply to workflow pods. | [optional]
**volumes** | Option<[**Vec<crate::models::Volume>**](Volume.md)> | Volumes is a list of volumes that can be mounted by containers in a template. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


