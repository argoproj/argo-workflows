# IoArgoprojEventsV1alpha1KafkaTrigger

KafkaTrigger refers to the specification of the Kafka trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compress** | **bool** |  | [optional] 
**flush_frequency** | **int** |  | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Parameters is the list of parameters that is applied to resolved Kafka trigger object. | [optional] 
**partition** | **int** |  | [optional] 
**partitioning_key** | **str** | The partitioning key for the messages put on the Kafka topic. +optional. | [optional] 
**payload** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional] 
**required_acks** | **int** | RequiredAcks used in producer to tell the broker how many replica acknowledgements Defaults to 1 (Only wait for the leader to ack). +optional. | [optional] 
**sasl** | [**IoArgoprojEventsV1alpha1SASLConfig**](IoArgoprojEventsV1alpha1SASLConfig.md) |  | [optional] 
**schema_registry** | [**IoArgoprojEventsV1alpha1SchemaRegistryConfig**](IoArgoprojEventsV1alpha1SchemaRegistryConfig.md) |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**topic** | **str** |  | [optional] 
**url** | **str** | URL of the Kafka broker, multiple URLs separated by comma. | [optional] 
**version** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_kafka_trigger import IoArgoprojEventsV1alpha1KafkaTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1KafkaTrigger from a JSON string
io_argoproj_events_v1alpha1_kafka_trigger_instance = IoArgoprojEventsV1alpha1KafkaTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1KafkaTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_kafka_trigger_dict = io_argoproj_events_v1alpha1_kafka_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1KafkaTrigger from a dict
io_argoproj_events_v1alpha1_kafka_trigger_form_dict = io_argoproj_events_v1alpha1_kafka_trigger.from_dict(io_argoproj_events_v1alpha1_kafka_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


