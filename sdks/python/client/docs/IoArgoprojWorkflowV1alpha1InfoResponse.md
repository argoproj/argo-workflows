# IoArgoprojWorkflowV1alpha1InfoResponse


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**columns** | [**List[IoArgoprojWorkflowV1alpha1Column]**](IoArgoprojWorkflowV1alpha1Column.md) |  | [optional] 
**links** | [**List[IoArgoprojWorkflowV1alpha1Link]**](IoArgoprojWorkflowV1alpha1Link.md) |  | [optional] 
**managed_namespace** | **str** |  | [optional] 
**modals** | **Dict[str, bool]** |  | [optional] 
**nav_color** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_info_response import IoArgoprojWorkflowV1alpha1InfoResponse

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1InfoResponse from a JSON string
io_argoproj_workflow_v1alpha1_info_response_instance = IoArgoprojWorkflowV1alpha1InfoResponse.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1InfoResponse.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_info_response_dict = io_argoproj_workflow_v1alpha1_info_response_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1InfoResponse from a dict
io_argoproj_workflow_v1alpha1_info_response_form_dict = io_argoproj_workflow_v1alpha1_info_response.from_dict(io_argoproj_workflow_v1alpha1_info_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


