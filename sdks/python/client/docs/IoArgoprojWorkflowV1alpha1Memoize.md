# IoArgoprojWorkflowV1alpha1Memoize

Memoization enables caching for the Outputs of the template

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cache** | [**IoArgoprojWorkflowV1alpha1Cache**](IoArgoprojWorkflowV1alpha1Cache.md) |  | 
**key** | **str** | Key is the key to use as the caching key | 
**max_age** | **str** | MaxAge is the maximum age (e.g. \&quot;180s\&quot;, \&quot;24h\&quot;) of an entry that is still considered valid. If an entry is older than the MaxAge, it will be ignored. | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_memoize import IoArgoprojWorkflowV1alpha1Memoize

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Memoize from a JSON string
io_argoproj_workflow_v1alpha1_memoize_instance = IoArgoprojWorkflowV1alpha1Memoize.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Memoize.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_memoize_dict = io_argoproj_workflow_v1alpha1_memoize_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Memoize from a dict
io_argoproj_workflow_v1alpha1_memoize_form_dict = io_argoproj_workflow_v1alpha1_memoize.from_dict(io_argoproj_workflow_v1alpha1_memoize_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


