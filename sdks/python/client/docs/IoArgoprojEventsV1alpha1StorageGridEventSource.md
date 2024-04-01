# IoArgoprojEventsV1alpha1StorageGridEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_url** | **str** | APIURL is the url of the storagegrid api. | [optional] 
**auth_token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**bucket** | **str** | Name of the bucket to register notifications for. | [optional] 
**events** | **List[str]** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1StorageGridFilter**](IoArgoprojEventsV1alpha1StorageGridFilter.md) |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**region** | **str** |  | [optional] 
**topic_arn** | **str** |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_storage_grid_event_source import IoArgoprojEventsV1alpha1StorageGridEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1StorageGridEventSource from a JSON string
io_argoproj_events_v1alpha1_storage_grid_event_source_instance = IoArgoprojEventsV1alpha1StorageGridEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1StorageGridEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_storage_grid_event_source_dict = io_argoproj_events_v1alpha1_storage_grid_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1StorageGridEventSource from a dict
io_argoproj_events_v1alpha1_storage_grid_event_source_form_dict = io_argoproj_events_v1alpha1_storage_grid_event_source.from_dict(io_argoproj_events_v1alpha1_storage_grid_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


