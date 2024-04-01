# IoArgoprojWorkflowV1alpha1NodeFlag


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**hooked** | **bool** | Hooked tracks whether or not this node was triggered by hook or onExit | [optional] 
**retried** | **bool** | Retried tracks whether or not this node was retried by retryStrategy | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_node_flag import IoArgoprojWorkflowV1alpha1NodeFlag

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1NodeFlag from a JSON string
io_argoproj_workflow_v1alpha1_node_flag_instance = IoArgoprojWorkflowV1alpha1NodeFlag.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1NodeFlag.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_node_flag_dict = io_argoproj_workflow_v1alpha1_node_flag_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1NodeFlag from a dict
io_argoproj_workflow_v1alpha1_node_flag_form_dict = io_argoproj_workflow_v1alpha1_node_flag.from_dict(io_argoproj_workflow_v1alpha1_node_flag_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


