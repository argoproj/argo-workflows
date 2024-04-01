# IoArgoprojEventsV1alpha1KafkaEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config** | **str** | Yaml format Sarama config for Kafka connection. It follows the struct of sarama.Config. See https://github.com/IBM/sarama/blob/main/config.go e.g.  consumer:   fetch:     min: 1 net:   MaxOpenRequests: 5  +optional | [optional] 
**connection_backoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  | [optional] 
**consumer_group** | [**IoArgoprojEventsV1alpha1KafkaConsumerGroup**](IoArgoprojEventsV1alpha1KafkaConsumerGroup.md) |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**limit_events_per_second** | **str** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**partition** | **str** |  | [optional] 
**sasl** | [**IoArgoprojEventsV1alpha1SASLConfig**](IoArgoprojEventsV1alpha1SASLConfig.md) |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**topic** | **str** |  | [optional] 
**url** | **str** |  | [optional] 
**version** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_kafka_event_source import IoArgoprojEventsV1alpha1KafkaEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1KafkaEventSource from a JSON string
io_argoproj_events_v1alpha1_kafka_event_source_instance = IoArgoprojEventsV1alpha1KafkaEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1KafkaEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_kafka_event_source_dict = io_argoproj_events_v1alpha1_kafka_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1KafkaEventSource from a dict
io_argoproj_events_v1alpha1_kafka_event_source_form_dict = io_argoproj_events_v1alpha1_kafka_event_source.from_dict(io_argoproj_events_v1alpha1_kafka_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


