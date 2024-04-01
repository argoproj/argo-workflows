# IoArgoprojEventsV1alpha1AMQPExchangeDeclareConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auto_delete** | **bool** |  | [optional] 
**durable** | **bool** |  | [optional] 
**internal** | **bool** |  | [optional] 
**no_wait** | **bool** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_amqp_exchange_declare_config import IoArgoprojEventsV1alpha1AMQPExchangeDeclareConfig

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1AMQPExchangeDeclareConfig from a JSON string
io_argoproj_events_v1alpha1_amqp_exchange_declare_config_instance = IoArgoprojEventsV1alpha1AMQPExchangeDeclareConfig.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1AMQPExchangeDeclareConfig.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_amqp_exchange_declare_config_dict = io_argoproj_events_v1alpha1_amqp_exchange_declare_config_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1AMQPExchangeDeclareConfig from a dict
io_argoproj_events_v1alpha1_amqp_exchange_declare_config_form_dict = io_argoproj_events_v1alpha1_amqp_exchange_declare_config.from_dict(io_argoproj_events_v1alpha1_amqp_exchange_declare_config_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


