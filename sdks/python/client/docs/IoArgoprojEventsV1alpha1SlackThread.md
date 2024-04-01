# IoArgoprojEventsV1alpha1SlackThread


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**broadcast_message_to_channel** | **bool** |  | [optional] 
**message_aggregation_key** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_slack_thread import IoArgoprojEventsV1alpha1SlackThread

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1SlackThread from a JSON string
io_argoproj_events_v1alpha1_slack_thread_instance = IoArgoprojEventsV1alpha1SlackThread.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1SlackThread.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_slack_thread_dict = io_argoproj_events_v1alpha1_slack_thread_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1SlackThread from a dict
io_argoproj_events_v1alpha1_slack_thread_form_dict = io_argoproj_events_v1alpha1_slack_thread.from_dict(io_argoproj_events_v1alpha1_slack_thread_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


