# IoArgoprojWorkflowV1alpha1Backoff

Backoff is a backoff strategy to use within retryStrategy

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**duration** | **str** | Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. \&quot;2m\&quot;, \&quot;1h\&quot;) | [optional] 
**factor** | **str** |  | [optional] 
**max_duration** | **str** | MaxDuration is the maximum amount of time allowed for a workflow in the backoff strategy | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_backoff import IoArgoprojWorkflowV1alpha1Backoff

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1Backoff from a JSON string
io_argoproj_workflow_v1alpha1_backoff_instance = IoArgoprojWorkflowV1alpha1Backoff.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1Backoff.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_backoff_dict = io_argoproj_workflow_v1alpha1_backoff_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1Backoff from a dict
io_argoproj_workflow_v1alpha1_backoff_form_dict = io_argoproj_workflow_v1alpha1_backoff.from_dict(io_argoproj_workflow_v1alpha1_backoff_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


