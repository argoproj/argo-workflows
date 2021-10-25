# V1alpha1NodeStatus

NodeStatus contains status information about an individual node in the workflow
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**boundary_id** | **str** | BoundaryID indicates the node ID of the associated template root node in which this node belongs to | [optional] 
**children** | **list[str]** | Children is a list of child node IDs | [optional] 
**daemoned** | **bool** | Daemoned tracks whether or not this node was daemoned and need to be terminated | [optional] 
**display_name** | **str** | DisplayName is a human readable representation of the node. Unique within a template boundary | [optional] 
**estimated_duration** | **int** | EstimatedDuration in seconds. | [optional] 
**finished_at** | **datetime** | Time at which this node completed | [optional] 
**host_node_name** | **str** | HostNodeName name of the Kubernetes node on which the Pod is running, if applicable | [optional] 
**id** | **str** | ID is a unique identifier of a node within the worklow It is implemented as a hash of the node name, which makes the ID deterministic | 
**inputs** | [**V1alpha1Inputs**](V1alpha1Inputs.md) |  | [optional] 
**memoization_status** | [**V1alpha1MemoizationStatus**](V1alpha1MemoizationStatus.md) |  | [optional] 
**message** | **str** | A human readable message indicating details about why the node is in this condition. | [optional] 
**name** | **str** | Name is unique name in the node tree used to generate the node ID | 
**outbound_nodes** | **list[str]** | OutboundNodes tracks the node IDs which are considered \&quot;outbound\&quot; nodes to a template invocation. For every invocation of a template, there are nodes which we considered as \&quot;outbound\&quot;. Essentially, these are last nodes in the execution sequence to run, before the template is considered completed. These nodes are then connected as parents to a following step.  In the case of single pod steps (i.e. container, script, resource templates), this list will be nil since the pod itself is already considered the \&quot;outbound\&quot; node. In the case of DAGs, outbound nodes are the \&quot;target\&quot; tasks (tasks with no children). In the case of steps, outbound nodes are all the containers involved in the last step group. NOTE: since templates are composable, the list of outbound nodes are carried upwards when a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of a template, will be a superset of the outbound nodes of its last children. | [optional] 
**outputs** | [**V1alpha1Outputs**](V1alpha1Outputs.md) |  | [optional] 
**phase** | **str** | Phase a simple, high-level summary of where the node is in its lifecycle. Can be used as a state machine. | [optional] 
**pod_ip** | **str** | PodIP captures the IP of the pod for daemoned steps | [optional] 
**progress** | **str** | Progress to completion | [optional] 
**resources_duration** | **dict(str, int)** | ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes. | [optional] 
**started_at** | **datetime** | Time at which this node started | [optional] 
**stored_template_id** | **str** | StoredTemplateID is the ID of stored template. DEPRECATED: This value is not used anymore. | [optional] 
**synchronization_status** | [**V1alpha1NodeSynchronizationStatus**](V1alpha1NodeSynchronizationStatus.md) |  | [optional] 
**template_name** | **str** | TemplateName is the template name which this node corresponds to. Not applicable to virtual nodes (e.g. Retry, StepGroup) | [optional] 
**template_ref** | [**V1alpha1TemplateRef**](V1alpha1TemplateRef.md) |  | [optional] 
**template_scope** | **str** | TemplateScope is the template scope in which the template of this node was retrieved. | [optional] 
**type** | **str** | Type indicates type of node | 
**workflow_template_name** | **str** | WorkflowTemplateName is the WorkflowTemplate resource name on which the resolved template of this node is retrieved. DEPRECATED: This value is not used anymore. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


