# IoArgoprojEventsV1alpha1EmitterEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**broker** | **str** | Broker URI to connect to. | [optional] 
**channel_key** | **str** |  | [optional] 
**channel_name** | **str** |  | [optional] 
**connection_backoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**password** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**username** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_emitter_event_source import IoArgoprojEventsV1alpha1EmitterEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1EmitterEventSource from a JSON string
io_argoproj_events_v1alpha1_emitter_event_source_instance = IoArgoprojEventsV1alpha1EmitterEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1EmitterEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_emitter_event_source_dict = io_argoproj_events_v1alpha1_emitter_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1EmitterEventSource from a dict
io_argoproj_events_v1alpha1_emitter_event_source_form_dict = io_argoproj_events_v1alpha1_emitter_event_source.from_dict(io_argoproj_events_v1alpha1_emitter_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


