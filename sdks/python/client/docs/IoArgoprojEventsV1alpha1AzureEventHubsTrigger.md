# IoArgoprojEventsV1alpha1AzureEventHubsTrigger


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fqdn** | **str** |  | [optional] 
**hub_name** | **str** |  | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**payload** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional] 
**shared_access_key** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**shared_access_key_name** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_azure_event_hubs_trigger import IoArgoprojEventsV1alpha1AzureEventHubsTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1AzureEventHubsTrigger from a JSON string
io_argoproj_events_v1alpha1_azure_event_hubs_trigger_instance = IoArgoprojEventsV1alpha1AzureEventHubsTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1AzureEventHubsTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_azure_event_hubs_trigger_dict = io_argoproj_events_v1alpha1_azure_event_hubs_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1AzureEventHubsTrigger from a dict
io_argoproj_events_v1alpha1_azure_event_hubs_trigger_form_dict = io_argoproj_events_v1alpha1_azure_event_hubs_trigger.from_dict(io_argoproj_events_v1alpha1_azure_event_hubs_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


