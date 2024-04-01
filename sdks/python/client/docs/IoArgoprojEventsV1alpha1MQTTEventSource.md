# IoArgoprojEventsV1alpha1MQTTEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**IoArgoprojEventsV1alpha1BasicAuth**](IoArgoprojEventsV1alpha1BasicAuth.md) |  | [optional] 
**client_id** | **str** |  | [optional] 
**connection_backoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**topic** | **str** |  | [optional] 
**url** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_mqtt_event_source import IoArgoprojEventsV1alpha1MQTTEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1MQTTEventSource from a JSON string
io_argoproj_events_v1alpha1_mqtt_event_source_instance = IoArgoprojEventsV1alpha1MQTTEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1MQTTEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_mqtt_event_source_dict = io_argoproj_events_v1alpha1_mqtt_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1MQTTEventSource from a dict
io_argoproj_events_v1alpha1_mqtt_event_source_form_dict = io_argoproj_events_v1alpha1_mqtt_event_source.from_dict(io_argoproj_events_v1alpha1_mqtt_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


