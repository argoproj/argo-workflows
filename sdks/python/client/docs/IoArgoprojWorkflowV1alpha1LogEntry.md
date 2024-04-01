# IoArgoprojWorkflowV1alpha1LogEntry


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**content** | **str** |  | [optional] 
**pod_name** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_log_entry import IoArgoprojWorkflowV1alpha1LogEntry

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1LogEntry from a JSON string
io_argoproj_workflow_v1alpha1_log_entry_instance = IoArgoprojWorkflowV1alpha1LogEntry.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1LogEntry.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_log_entry_dict = io_argoproj_workflow_v1alpha1_log_entry_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1LogEntry from a dict
io_argoproj_workflow_v1alpha1_log_entry_form_dict = io_argoproj_workflow_v1alpha1_log_entry.from_dict(io_argoproj_workflow_v1alpha1_log_entry_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


