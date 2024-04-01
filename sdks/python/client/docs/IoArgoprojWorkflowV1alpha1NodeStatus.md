# IoArgoprojWorkflowV1alpha1NodeStatus

NodeStatus contains status information about an individual node in the workflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**boundary_id** | **str** | BoundaryID indicates the node ID of the associated template root node in which this node belongs to | [optional] 
**children** | **List[str]** | Children is a list of child node IDs | [optional] 
**daemoned** | **bool** | Daemoned tracks whether or not this node was daemoned and need to be terminated | [optional] 
**display_name** | **str** | DisplayName is a human readable representation of the node. Unique within a template boundary | [optional] 
**estimated_duration** | **int** | EstimatedDuration in seconds. | [optional] 
**finished_at** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**host_node_name** | **str** | HostNodeName name of the Kubernetes node on which the Pod is running, if applicable | [optional] 
**id** | **str** | ID is a unique identifier of a node within the worklow It is implemented as a hash of the node name, which makes the ID deterministic | 
**inputs** | [**IoArgoprojWorkflowV1alpha1Inputs**](IoArgoprojWorkflowV1alpha1Inputs.md) |  | [optional] 
**memoization_status** | [**IoArgoprojWorkflowV1alpha1MemoizationStatus**](IoArgoprojWorkflowV1alpha1MemoizationStatus.md) |  | [optional] 
**message** | **str** | A human readable message indicating details about why the node is in this condition. | [optional] 
**name** | **str** | Name is unique name in the node tree used to generate the node ID | 
**node_flag** | [**IoArgoprojWorkflowV1alpha1NodeFlag**](IoArgoprojWorkflowV1alpha1NodeFlag.md) |  | [optional] 
**outbound_nodes** | **List[str]** | OutboundNodes tracks the node IDs which are considered \&quot;outbound\&quot; nodes to a template invocation. For every invocation of a template, there are nodes which we considered as \&quot;outbound\&quot;. Essentially, these are last nodes in the execution sequence to run, before the template is considered completed. These nodes are then connected as parents to a following step.  In the case of single pod steps (i.e. container, script, resource templates), this list will be nil since the pod itself is already considered the \&quot;outbound\&quot; node. In the case of DAGs, outbound nodes are the \&quot;target\&quot; tasks (tasks with no children). In the case of steps, outbound nodes are all the containers involved in the last step group. NOTE: since templates are composable, the list of outbound nodes are carried upwards when a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of a template, will be a superset of the outbound nodes of its last children. | [optional] 
**outputs** | [**IoArgoprojWorkflowV1alpha1Outputs**](IoArgoprojWorkflowV1alpha1Outputs.md) |  | [optional] 
**phase** | **str** | Phase a simple, high-level summary of where the node is in its lifecycle. Can be used as a state machine. Will be one of these values \&quot;Pending\&quot;, \&quot;Running\&quot; before the node is completed, or \&quot;Succeeded\&quot;, \&quot;Skipped\&quot;, \&quot;Failed\&quot;, \&quot;Error\&quot;, or \&quot;Omitted\&quot; as a final state. | [optional] 
**pod_ip** | **str** | PodIP captures the IP of the pod for daemoned steps | [optional] 
**progress** | **str** | Progress to completion | [optional] 
**resources_duration** | **Dict[str, int]** | ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes. | [optional] 
**started_at** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**synchronization_status** | [**IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus**](IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus.md) |  | [optional] 
**template_name** | **str** | TemplateName is the template name which this node corresponds to. Not applicable to virtual nodes (e.g. Retry, StepGroup) | [optional] 
**template_ref** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  | [optional] 
**template_scope** | **str** | TemplateScope is the template scope in which the template of this node was retrieved. | [optional] 
**type** | **str** | Type indicates type of node | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_node_status import IoArgoprojWorkflowV1alpha1NodeStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1NodeStatus from a JSON string
io_argoproj_workflow_v1alpha1_node_status_instance = IoArgoprojWorkflowV1alpha1NodeStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1NodeStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_node_status_dict = io_argoproj_workflow_v1alpha1_node_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1NodeStatus from a dict
io_argoproj_workflow_v1alpha1_node_status_form_dict = io_argoproj_workflow_v1alpha1_node_status.from_dict(io_argoproj_workflow_v1alpha1_node_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


