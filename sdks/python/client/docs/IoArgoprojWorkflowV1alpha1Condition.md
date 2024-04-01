# IoArgoprojWorkflowV1alpha1Condition


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**message** | **str** | Message is the condition message | [optional] 
**status** | **str** | Status is the status of the condition | [optional] 
**type** | **str** | Type is the type of condition | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_condition import IoArgoprojWorkflowV1alpha1Condition

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Condition from a JSON string
io_argoproj_workflow_v1alpha1_condition_instance = IoArgoprojWorkflowV1alpha1Condition.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Condition.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_condition_dict = io_argoproj_workflow_v1alpha1_condition_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Condition from a dict
io_argoproj_workflow_v1alpha1_condition_form_dict = io_argoproj_workflow_v1alpha1_condition.from_dict(io_argoproj_workflow_v1alpha1_condition_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


