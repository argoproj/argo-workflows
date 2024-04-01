# IoArgoprojWorkflowV1alpha1SemaphoreStatus


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**holding** | [**List[IoArgoprojWorkflowV1alpha1SemaphoreHolding]**](IoArgoprojWorkflowV1alpha1SemaphoreHolding.md) | Holding stores the list of resource acquired synchronization lock for workflows. | [optional] 
**waiting** | [**List[IoArgoprojWorkflowV1alpha1SemaphoreHolding]**](IoArgoprojWorkflowV1alpha1SemaphoreHolding.md) | Waiting indicates the list of current synchronization lock holders. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_semaphore_status import IoArgoprojWorkflowV1alpha1SemaphoreStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1SemaphoreStatus from a JSON string
io_argoproj_workflow_v1alpha1_semaphore_status_instance = IoArgoprojWorkflowV1alpha1SemaphoreStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1SemaphoreStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_semaphore_status_dict = io_argoproj_workflow_v1alpha1_semaphore_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1SemaphoreStatus from a dict
io_argoproj_workflow_v1alpha1_semaphore_status_form_dict = io_argoproj_workflow_v1alpha1_semaphore_status.from_dict(io_argoproj_workflow_v1alpha1_semaphore_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


