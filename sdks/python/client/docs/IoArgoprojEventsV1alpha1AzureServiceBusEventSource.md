# IoArgoprojEventsV1alpha1AzureServiceBusEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**connection_string** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**fully_qualified_namespace** | **str** |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**queue_name** | **str** |  | [optional] 
**subscription_name** | **str** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**topic_name** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_azure_service_bus_event_source import IoArgoprojEventsV1alpha1AzureServiceBusEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1AzureServiceBusEventSource from a JSON string
io_argoproj_events_v1alpha1_azure_service_bus_event_source_instance = IoArgoprojEventsV1alpha1AzureServiceBusEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1AzureServiceBusEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_azure_service_bus_event_source_dict = io_argoproj_events_v1alpha1_azure_service_bus_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1AzureServiceBusEventSource from a dict
io_argoproj_events_v1alpha1_azure_service_bus_event_source_form_dict = io_argoproj_events_v1alpha1_azure_service_bus_event_source.from_dict(io_argoproj_events_v1alpha1_azure_service_bus_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


