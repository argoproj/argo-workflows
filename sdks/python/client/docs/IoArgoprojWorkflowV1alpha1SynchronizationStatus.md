# IoArgoprojWorkflowV1alpha1SynchronizationStatus

SynchronizationStatus stores the status of semaphore and mutex.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**mutex** | [**IoArgoprojWorkflowV1alpha1MutexStatus**](IoArgoprojWorkflowV1alpha1MutexStatus.md) |  | [optional] 
**semaphore** | [**IoArgoprojWorkflowV1alpha1SemaphoreStatus**](IoArgoprojWorkflowV1alpha1SemaphoreStatus.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_synchronization_status import IoArgoprojWorkflowV1alpha1SynchronizationStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1SynchronizationStatus from a JSON string
io_argoproj_workflow_v1alpha1_synchronization_status_instance = IoArgoprojWorkflowV1alpha1SynchronizationStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1SynchronizationStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_synchronization_status_dict = io_argoproj_workflow_v1alpha1_synchronization_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1SynchronizationStatus from a dict
io_argoproj_workflow_v1alpha1_synchronization_status_form_dict = io_argoproj_workflow_v1alpha1_synchronization_status.from_dict(io_argoproj_workflow_v1alpha1_synchronization_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


