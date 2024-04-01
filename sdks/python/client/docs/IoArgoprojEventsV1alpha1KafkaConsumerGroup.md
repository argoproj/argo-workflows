# IoArgoprojEventsV1alpha1KafkaConsumerGroup


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**group_name** | **str** |  | [optional] 
**oldest** | **bool** |  | [optional] 
**rebalance_strategy** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_kafka_consumer_group import IoArgoprojEventsV1alpha1KafkaConsumerGroup

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1KafkaConsumerGroup from a JSON string
io_argoproj_events_v1alpha1_kafka_consumer_group_instance = IoArgoprojEventsV1alpha1KafkaConsumerGroup.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1KafkaConsumerGroup.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_kafka_consumer_group_dict = io_argoproj_events_v1alpha1_kafka_consumer_group_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1KafkaConsumerGroup from a dict
io_argoproj_events_v1alpha1_kafka_consumer_group_form_dict = io_argoproj_events_v1alpha1_kafka_consumer_group.from_dict(io_argoproj_events_v1alpha1_kafka_consumer_group_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


