# IoArgoprojEventsV1alpha1WebhookEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**webhook_context** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_webhook_event_source import IoArgoprojEventsV1alpha1WebhookEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1WebhookEventSource from a JSON string
io_argoproj_events_v1alpha1_webhook_event_source_instance = IoArgoprojEventsV1alpha1WebhookEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1WebhookEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_webhook_event_source_dict = io_argoproj_events_v1alpha1_webhook_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1WebhookEventSource from a dict
io_argoproj_events_v1alpha1_webhook_event_source_form_dict = io_argoproj_events_v1alpha1_webhook_event_source.from_dict(io_argoproj_events_v1alpha1_webhook_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


