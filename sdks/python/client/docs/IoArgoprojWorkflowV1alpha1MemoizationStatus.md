# IoArgoprojWorkflowV1alpha1MemoizationStatus

MemoizationStatus is the status of this memoized node

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cache_name** | **str** | Cache is the name of the cache that was used | 
**hit** | **bool** | Hit indicates whether this node was created from a cache entry | 
**key** | **str** | Key is the name of the key used for this node&#39;s cache | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_memoization_status import IoArgoprojWorkflowV1alpha1MemoizationStatus

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1MemoizationStatus from a JSON string
io_argoproj_workflow_v1alpha1_memoization_status_instance = IoArgoprojWorkflowV1alpha1MemoizationStatus.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1MemoizationStatus.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_memoization_status_dict = io_argoproj_workflow_v1alpha1_memoization_status_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1MemoizationStatus from a dict
io_argoproj_workflow_v1alpha1_memoization_status_form_dict = io_argoproj_workflow_v1alpha1_memoization_status.from_dict(io_argoproj_workflow_v1alpha1_memoization_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


