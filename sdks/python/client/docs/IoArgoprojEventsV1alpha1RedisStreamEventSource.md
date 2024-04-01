# IoArgoprojEventsV1alpha1RedisStreamEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**consumer_group** | **str** |  | [optional] 
**db** | **int** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**host_address** | **str** |  | [optional] 
**max_msg_count_per_read** | **int** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**password** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**streams** | **List[str]** | Streams to look for entries. XREADGROUP is used on all streams using a single consumer group. | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**username** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_redis_stream_event_source import IoArgoprojEventsV1alpha1RedisStreamEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1RedisStreamEventSource from a JSON string
io_argoproj_events_v1alpha1_redis_stream_event_source_instance = IoArgoprojEventsV1alpha1RedisStreamEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1RedisStreamEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_redis_stream_event_source_dict = io_argoproj_events_v1alpha1_redis_stream_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1RedisStreamEventSource from a dict
io_argoproj_events_v1alpha1_redis_stream_event_source_form_dict = io_argoproj_events_v1alpha1_redis_stream_event_source.from_dict(io_argoproj_events_v1alpha1_redis_stream_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


