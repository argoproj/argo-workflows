# IoArgoprojWorkflowV1alpha1NodeResult


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** |  | [optional] 
**outputs** | [**IoArgoprojWorkflowV1alpha1Outputs**](IoArgoprojWorkflowV1alpha1Outputs.md) |  | [optional] 
**phase** | **str** |  | [optional] 
**progress** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_node_result import IoArgoprojWorkflowV1alpha1NodeResult

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1NodeResult from a JSON string
io_argoproj_workflow_v1alpha1_node_result_instance = IoArgoprojWorkflowV1alpha1NodeResult.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1NodeResult.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_node_result_dict = io_argoproj_workflow_v1alpha1_node_result_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1NodeResult from a dict
io_argoproj_workflow_v1alpha1_node_result_form_dict = io_argoproj_workflow_v1alpha1_node_result.from_dict(io_argoproj_workflow_v1alpha1_node_result_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


