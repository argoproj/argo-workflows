# IoArgoprojEventsV1alpha1NSQEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**channel** | **str** |  | [optional] 
**connection_backoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**host_address** | **str** |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**topic** | **str** | Topic to subscribe to. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_nsq_event_source import IoArgoprojEventsV1alpha1NSQEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1NSQEventSource from a JSON string
io_argoproj_events_v1alpha1_nsq_event_source_instance = IoArgoprojEventsV1alpha1NSQEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1NSQEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_nsq_event_source_dict = io_argoproj_events_v1alpha1_nsq_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1NSQEventSource from a dict
io_argoproj_events_v1alpha1_nsq_event_source_form_dict = io_argoproj_events_v1alpha1_nsq_event_source.from_dict(io_argoproj_events_v1alpha1_nsq_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


