# IoArgoprojWorkflowV1alpha1Cache

Cache is the configuration for the type of cache to be used

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map** | [**ConfigMapKeySelector**](ConfigMapKeySelector.md) |  | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cache import IoArgoprojWorkflowV1alpha1Cache

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Cache from a JSON string
io_argoproj_workflow_v1alpha1_cache_instance = IoArgoprojWorkflowV1alpha1Cache.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Cache.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_cache_dict = io_argoproj_workflow_v1alpha1_cache_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Cache from a dict
io_argoproj_workflow_v1alpha1_cache_form_dict = io_argoproj_workflow_v1alpha1_cache.from_dict(io_argoproj_workflow_v1alpha1_cache_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


