# IoArgoprojEventsV1alpha1SlackEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**signing_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_slack_event_source import IoArgoprojEventsV1alpha1SlackEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1SlackEventSource from a JSON string
io_argoproj_events_v1alpha1_slack_event_source_instance = IoArgoprojEventsV1alpha1SlackEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1SlackEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_slack_event_source_dict = io_argoproj_events_v1alpha1_slack_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1SlackEventSource from a dict
io_argoproj_events_v1alpha1_slack_event_source_form_dict = io_argoproj_events_v1alpha1_slack_event_source.from_dict(io_argoproj_events_v1alpha1_slack_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


