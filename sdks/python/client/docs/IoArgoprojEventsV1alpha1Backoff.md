# IoArgoprojEventsV1alpha1Backoff


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**duration** | [**IoArgoprojEventsV1alpha1Int64OrString**](IoArgoprojEventsV1alpha1Int64OrString.md) |  | [optional] 
**factor** | [**IoArgoprojEventsV1alpha1Amount**](IoArgoprojEventsV1alpha1Amount.md) |  | [optional] 
**jitter** | [**IoArgoprojEventsV1alpha1Amount**](IoArgoprojEventsV1alpha1Amount.md) |  | [optional] 
**steps** | **int** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_backoff import IoArgoprojEventsV1alpha1Backoff

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1Backoff from a JSON string
io_argoproj_events_v1alpha1_backoff_instance = IoArgoprojEventsV1alpha1Backoff.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1Backoff.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_backoff_dict = io_argoproj_events_v1alpha1_backoff_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1Backoff from a dict
io_argoproj_events_v1alpha1_backoff_form_dict = io_argoproj_events_v1alpha1_backoff.from_dict(io_argoproj_events_v1alpha1_backoff_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


