# IoArgoprojWorkflowV1alpha1MutexStatus

MutexStatus contains which objects hold  mutex locks, and which objects this workflow is waiting on to release locks.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**holding** | [**List[IoArgoprojWorkflowV1alpha1MutexHolding]**](IoArgoprojWorkflowV1alpha1MutexHolding.md) | Holding is a list of mutexes and their respective objects that are held by mutex lock for this io.argoproj.workflow.v1alpha1. | [optional] 
**waiting** | [**List[IoArgoprojWorkflowV1alpha1MutexHolding]**](IoArgoprojWorkflowV1alpha1MutexHolding.md) | Waiting is a list of mutexes and their respective objects this workflow is waiting for. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_mutex_status import IoArgoprojWorkflowV1alpha1MutexStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1MutexStatus from a JSON string
io_argoproj_workflow_v1alpha1_mutex_status_instance = IoArgoprojWorkflowV1alpha1MutexStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1MutexStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_mutex_status_dict = io_argoproj_workflow_v1alpha1_mutex_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1MutexStatus from a dict
io_argoproj_workflow_v1alpha1_mutex_status_form_dict = io_argoproj_workflow_v1alpha1_mutex_status.from_dict(io_argoproj_workflow_v1alpha1_mutex_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


