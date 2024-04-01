# IoArgoprojEventsV1alpha1RedisEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**channels** | **List[str]** |  | [optional] 
**db** | **int** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**host_address** | **str** |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**namespace** | **str** |  | [optional] 
**password** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**username** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_redis_event_source import IoArgoprojEventsV1alpha1RedisEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1RedisEventSource from a JSON string
io_argoproj_events_v1alpha1_redis_event_source_instance = IoArgoprojEventsV1alpha1RedisEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1RedisEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_redis_event_source_dict = io_argoproj_events_v1alpha1_redis_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1RedisEventSource from a dict
io_argoproj_events_v1alpha1_redis_event_source_form_dict = io_argoproj_events_v1alpha1_redis_event_source.from_dict(io_argoproj_events_v1alpha1_redis_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


