# IoArgoprojEventsV1alpha1GenericEventSource

GenericEventSource refers to a generic event source. It can be used to implement a custom event source.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**config** | **str** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**insecure** | **bool** | Insecure determines the type of connection. | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**url** | **str** | URL of the gRPC server that implements the event source. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_generic_event_source import IoArgoprojEventsV1alpha1GenericEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1GenericEventSource from a JSON string
io_argoproj_events_v1alpha1_generic_event_source_instance = IoArgoprojEventsV1alpha1GenericEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1GenericEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_generic_event_source_dict = io_argoproj_events_v1alpha1_generic_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1GenericEventSource from a dict
io_argoproj_events_v1alpha1_generic_event_source_form_dict = io_argoproj_events_v1alpha1_generic_event_source.from_dict(io_argoproj_events_v1alpha1_generic_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


