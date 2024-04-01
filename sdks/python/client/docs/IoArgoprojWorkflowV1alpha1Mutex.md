# IoArgoprojWorkflowV1alpha1Mutex

Mutex holds Mutex configuration

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | name of the mutex | [optional] 
**namespace** | **str** | Namespace is the namespace of the mutex, default: [namespace of workflow] | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_mutex import IoArgoprojWorkflowV1alpha1Mutex

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Mutex from a JSON string
io_argoproj_workflow_v1alpha1_mutex_instance = IoArgoprojWorkflowV1alpha1Mutex.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Mutex.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_mutex_dict = io_argoproj_workflow_v1alpha1_mutex_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Mutex from a dict
io_argoproj_workflow_v1alpha1_mutex_form_dict = io_argoproj_workflow_v1alpha1_mutex.from_dict(io_argoproj_workflow_v1alpha1_mutex_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


