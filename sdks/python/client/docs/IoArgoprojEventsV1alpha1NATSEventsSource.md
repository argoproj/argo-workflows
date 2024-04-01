# IoArgoprojEventsV1alpha1NATSEventsSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**IoArgoprojEventsV1alpha1NATSAuth**](IoArgoprojEventsV1alpha1NATSAuth.md) |  | [optional] 
**connection_backoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**subject** | **str** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**url** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_nats_events_source import IoArgoprojEventsV1alpha1NATSEventsSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1NATSEventsSource from a JSON string
io_argoproj_events_v1alpha1_nats_events_source_instance = IoArgoprojEventsV1alpha1NATSEventsSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1NATSEventsSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_nats_events_source_dict = io_argoproj_events_v1alpha1_nats_events_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1NATSEventsSource from a dict
io_argoproj_events_v1alpha1_nats_events_source_form_dict = io_argoproj_events_v1alpha1_nats_events_source.from_dict(io_argoproj_events_v1alpha1_nats_events_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


