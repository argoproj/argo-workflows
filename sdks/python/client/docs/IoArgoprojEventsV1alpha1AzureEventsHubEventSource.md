# IoArgoprojEventsV1alpha1AzureEventsHubEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**fqdn** | **str** |  | [optional] 
**hub_name** | **str** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**shared_access_key** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**shared_access_key_name** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_azure_events_hub_event_source import IoArgoprojEventsV1alpha1AzureEventsHubEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1AzureEventsHubEventSource from a JSON string
io_argoproj_events_v1alpha1_azure_events_hub_event_source_instance = IoArgoprojEventsV1alpha1AzureEventsHubEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1AzureEventsHubEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_azure_events_hub_event_source_dict = io_argoproj_events_v1alpha1_azure_events_hub_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1AzureEventsHubEventSource from a dict
io_argoproj_events_v1alpha1_azure_events_hub_event_source_form_dict = io_argoproj_events_v1alpha1_azure_events_hub_event_source.from_dict(io_argoproj_events_v1alpha1_azure_events_hub_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


