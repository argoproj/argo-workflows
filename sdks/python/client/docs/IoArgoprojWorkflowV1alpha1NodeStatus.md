# IoArgoprojWorkflowV1alpha1NodeStatus


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**boundary_id** | **str** |  | [optional] 
**children** | **[str]** |  | [optional] 
**daemoned** | **bool** |  | [optional] 
**display_name** | **str** |  | [optional] 
**estimated_duration** | **str** | EstimatedDuration in seconds. | [optional] 
**finished_at** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**host_node_name** | **str** |  | [optional] 
**id** | **str** |  | [optional] 
**inputs** | [**IoArgoprojWorkflowV1alpha1Inputs**](IoArgoprojWorkflowV1alpha1Inputs.md) |  | [optional] 
**memoization_status** | [**IoArgoprojWorkflowV1alpha1MemoizationStatus**](IoArgoprojWorkflowV1alpha1MemoizationStatus.md) |  | [optional] 
**message** | **str** | A human readable message indicating details about why the node is in this condition. | [optional] 
**name** | **str** |  | [optional] 
**outbound_nodes** | **[str]** | OutboundNodes tracks the node IDs which are considered \&quot;outbound\&quot; nodes to a template invocation. For every invocation of a template, there are nodes which we considered as \&quot;outbound\&quot;. Essentially, these are last nodes in the execution sequence to run, before the template is considered completed. These nodes are then connected as parents to a following step.  In the case of single pod steps (i.e. container, script, resource templates), this list will be nil since the pod itself is already considered the \&quot;outbound\&quot; node. In the case of DAGs, outbound nodes are the \&quot;target\&quot; tasks (tasks with no children). In the case of steps, outbound nodes are all the containers involved in the last step group. NOTE: since templates are composable, the list of outbound nodes are carried upwards when a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of a template, will be a superset of the outbound nodes of its last children. | [optional] 
**outputs** | [**IoArgoprojWorkflowV1alpha1Outputs**](IoArgoprojWorkflowV1alpha1Outputs.md) |  | [optional] 
**phase** | **str** | Phase a simple, high-level summary of where the node is in its lifecycle. Can be used as a state machine. | [optional] 
**pod_ip** | **str** |  | [optional] 
**progress** | **str** |  | [optional] 
**resources_duration** | **{str: (str,)}** | ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes. | [optional] 
**started_at** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**synchronization_status** | [**IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus**](IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus.md) |  | [optional] 
**template_name** | **str** |  | [optional] 
**template_ref** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  | [optional] 
**template_scope** | **str** | TemplateScope is the template scope in which the template of this node was retrieved. | [optional] 
**type** | **str** |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


