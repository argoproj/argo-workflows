# IoArgoprojEventsV1alpha1AzureServiceBusTrigger


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**connection_string** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**payload** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional] 
**queue_name** | **str** |  | [optional] 
**subscription_name** | **str** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**topic_name** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_azure_service_bus_trigger import IoArgoprojEventsV1alpha1AzureServiceBusTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1AzureServiceBusTrigger from a JSON string
io_argoproj_events_v1alpha1_azure_service_bus_trigger_instance = IoArgoprojEventsV1alpha1AzureServiceBusTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1AzureServiceBusTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_azure_service_bus_trigger_dict = io_argoproj_events_v1alpha1_azure_service_bus_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1AzureServiceBusTrigger from a dict
io_argoproj_events_v1alpha1_azure_service_bus_trigger_form_dict = io_argoproj_events_v1alpha1_azure_service_bus_trigger.from_dict(io_argoproj_events_v1alpha1_azure_service_bus_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


