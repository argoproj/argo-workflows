# IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus

NodeSynchronizationStatus stores the status of a node

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**waiting** | **str** | Waiting is the name of the lock that this node is waiting for | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_node_synchronization_status import IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus from a JSON string
io_argoproj_workflow_v1alpha1_node_synchronization_status_instance = IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_node_synchronization_status_dict = io_argoproj_workflow_v1alpha1_node_synchronization_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus from a dict
io_argoproj_workflow_v1alpha1_node_synchronization_status_form_dict = io_argoproj_workflow_v1alpha1_node_synchronization_status.from_dict(io_argoproj_workflow_v1alpha1_node_synchronization_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


