# IoArgoprojEventsV1alpha1AzureQueueStorageEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**connection_string** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**decode_message** | **bool** |  | [optional] 
**dlq** | **bool** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**queue_name** | **str** |  | [optional] 
**storage_account_name** | **str** |  | [optional] 
**wait_time_in_seconds** | **int** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_azure_queue_storage_event_source import IoArgoprojEventsV1alpha1AzureQueueStorageEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1AzureQueueStorageEventSource from a JSON string
io_argoproj_events_v1alpha1_azure_queue_storage_event_source_instance = IoArgoprojEventsV1alpha1AzureQueueStorageEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1AzureQueueStorageEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_azure_queue_storage_event_source_dict = io_argoproj_events_v1alpha1_azure_queue_storage_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1AzureQueueStorageEventSource from a dict
io_argoproj_events_v1alpha1_azure_queue_storage_event_source_form_dict = io_argoproj_events_v1alpha1_azure_queue_storage_event_source.from_dict(io_argoproj_events_v1alpha1_azure_queue_storage_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


