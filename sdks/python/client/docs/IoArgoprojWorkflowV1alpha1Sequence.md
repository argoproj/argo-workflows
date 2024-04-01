# IoArgoprojWorkflowV1alpha1Sequence

Sequence expands a workflow step into numeric range

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**count** | **str** |  | [optional] 
**end** | **str** |  | [optional] 
**format** | **str** | Format is a printf format string to format the value in the sequence | [optional] 
**start** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_sequence import IoArgoprojWorkflowV1alpha1Sequence

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Sequence from a JSON string
io_argoproj_workflow_v1alpha1_sequence_instance = IoArgoprojWorkflowV1alpha1Sequence.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Sequence.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_sequence_dict = io_argoproj_workflow_v1alpha1_sequence_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Sequence from a dict
io_argoproj_workflow_v1alpha1_sequence_form_dict = io_argoproj_workflow_v1alpha1_sequence.from_dict(io_argoproj_workflow_v1alpha1_sequence_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


