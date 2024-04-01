# IoArgoprojEventsV1alpha1AMQPConsumeConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auto_ack** | **bool** |  | [optional] 
**consumer_tag** | **str** |  | [optional] 
**exclusive** | **bool** |  | [optional] 
**no_local** | **bool** |  | [optional] 
**no_wait** | **bool** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_amqp_consume_config import IoArgoprojEventsV1alpha1AMQPConsumeConfig

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1AMQPConsumeConfig from a JSON string
io_argoproj_events_v1alpha1_amqp_consume_config_instance = IoArgoprojEventsV1alpha1AMQPConsumeConfig.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1AMQPConsumeConfig.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_amqp_consume_config_dict = io_argoproj_events_v1alpha1_amqp_consume_config_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1AMQPConsumeConfig from a dict
io_argoproj_events_v1alpha1_amqp_consume_config_form_dict = io_argoproj_events_v1alpha1_amqp_consume_config.from_dict(io_argoproj_events_v1alpha1_amqp_consume_config_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


