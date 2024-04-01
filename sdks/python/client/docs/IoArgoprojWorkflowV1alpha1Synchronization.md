# IoArgoprojWorkflowV1alpha1Synchronization

Synchronization holds synchronization lock configuration

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**mutex** | [**IoArgoprojWorkflowV1alpha1Mutex**](IoArgoprojWorkflowV1alpha1Mutex.md) |  | [optional] 
**semaphore** | [**IoArgoprojWorkflowV1alpha1SemaphoreRef**](IoArgoprojWorkflowV1alpha1SemaphoreRef.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_synchronization import IoArgoprojWorkflowV1alpha1Synchronization

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Synchronization from a JSON string
io_argoproj_workflow_v1alpha1_synchronization_instance = IoArgoprojWorkflowV1alpha1Synchronization.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Synchronization.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_synchronization_dict = io_argoproj_workflow_v1alpha1_synchronization_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Synchronization from a dict
io_argoproj_workflow_v1alpha1_synchronization_form_dict = io_argoproj_workflow_v1alpha1_synchronization.from_dict(io_argoproj_workflow_v1alpha1_synchronization_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


