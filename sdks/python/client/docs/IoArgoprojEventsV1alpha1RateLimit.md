# IoArgoprojEventsV1alpha1RateLimit


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**requests_per_unit** | **int** |  | [optional] 
**unit** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_rate_limit import IoArgoprojEventsV1alpha1RateLimit

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1RateLimit from a JSON string
io_argoproj_events_v1alpha1_rate_limit_instance = IoArgoprojEventsV1alpha1RateLimit.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1RateLimit.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_rate_limit_dict = io_argoproj_events_v1alpha1_rate_limit_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1RateLimit from a dict
io_argoproj_events_v1alpha1_rate_limit_form_dict = io_argoproj_events_v1alpha1_rate_limit.from_dict(io_argoproj_events_v1alpha1_rate_limit_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


