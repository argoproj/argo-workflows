# IoArgoprojWorkflowV1alpha1NodeStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**boundary_id** | Option<**String**> | BoundaryID indicates the node ID of the associated template root node in which this node belongs to | [optional]
**children** | Option<**Vec<String>**> | Children is a list of child node IDs | [optional]
**daemoned** | Option<**bool**> | Daemoned tracks whether or not this node was daemoned and need to be terminated | [optional]
**display_name** | Option<**String**> | DisplayName is a human readable representation of the node. Unique within a template boundary | [optional]
**estimated_duration** | Option<**i32**> | EstimatedDuration in seconds. | [optional]
**finished_at** | Option<**String**> | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional]
**host_node_name** | Option<**String**> | HostNodeName name of the Kubernetes node on which the Pod is running, if applicable | [optional]
**id** | **String** | ID is a unique identifier of a node within the worklow It is implemented as a hash of the node name, which makes the ID deterministic | 
**inputs** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Inputs**](io.argoproj.workflow.v1alpha1.Inputs.md)> |  | [optional]
**memoization_status** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1MemoizationStatus**](io.argoproj.workflow.v1alpha1.MemoizationStatus.md)> |  | [optional]
**message** | Option<**String**> | A human readable message indicating details about why the node is in this condition. | [optional]
**name** | **String** | Name is unique name in the node tree used to generate the node ID | 
**outbound_nodes** | Option<**Vec<String>**> | OutboundNodes tracks the node IDs which are considered \"outbound\" nodes to a template invocation. For every invocation of a template, there are nodes which we considered as \"outbound\". Essentially, these are last nodes in the execution sequence to run, before the template is considered completed. These nodes are then connected as parents to a following step.  In the case of single pod steps (i.e. container, script, resource templates), this list will be nil since the pod itself is already considered the \"outbound\" node. In the case of DAGs, outbound nodes are the \"target\" tasks (tasks with no children). In the case of steps, outbound nodes are all the containers involved in the last step group. NOTE: since templates are composable, the list of outbound nodes are carried upwards when a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of a template, will be a superset of the outbound nodes of its last children. | [optional]
**outputs** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Outputs**](io.argoproj.workflow.v1alpha1.Outputs.md)> |  | [optional]
**phase** | Option<**String**> | Phase a simple, high-level summary of where the node is in its lifecycle. Can be used as a state machine. Will be one of these values \"Pending\", \"Running\" before the node is completed, or \"Succeeded\", \"Skipped\", \"Failed\", \"Error\", or \"Omitted\" as a final state. | [optional]
**pod_ip** | Option<**String**> | PodIP captures the IP of the pod for daemoned steps | [optional]
**progress** | Option<**String**> | Progress to completion | [optional]
**resources_duration** | Option<**::std::collections::HashMap<String, i64>**> | ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes. | [optional]
**started_at** | Option<**String**> | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional]
**synchronization_status** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus**](io.argoproj.workflow.v1alpha1.NodeSynchronizationStatus.md)> |  | [optional]
**template_name** | Option<**String**> | TemplateName is the template name which this node corresponds to. Not applicable to virtual nodes (e.g. Retry, StepGroup) | [optional]
**template_ref** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1TemplateRef**](io.argoproj.workflow.v1alpha1.TemplateRef.md)> |  | [optional]
**template_scope** | Option<**String**> | TemplateScope is the template scope in which the template of this node was retrieved. | [optional]
**_type** | **String** | Type indicates type of node | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


