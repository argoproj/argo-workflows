# IoArgoprojEventsV1alpha1Status

Status is a common structure which can be used for Status field.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**conditions** | [**List[IoArgoprojEventsV1alpha1Condition]**](IoArgoprojEventsV1alpha1Condition.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_status import IoArgoprojEventsV1alpha1Status

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1Status from a JSON string
io_argoproj_events_v1alpha1_status_instance = IoArgoprojEventsV1alpha1Status.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1Status.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_status_dict = io_argoproj_events_v1alpha1_status_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1Status from a dict
io_argoproj_events_v1alpha1_status_form_dict = io_argoproj_events_v1alpha1_status.from_dict(io_argoproj_events_v1alpha1_status_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


