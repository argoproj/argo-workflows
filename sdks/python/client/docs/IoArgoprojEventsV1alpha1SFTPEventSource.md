# IoArgoprojEventsV1alpha1SFTPEventSource

SFTPEventSource describes an event-source for sftp related events.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**address** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**event_type** | **str** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**password** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**poll_interval_duration** | **str** |  | [optional] 
**ssh_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**username** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**watch_path_config** | [**IoArgoprojEventsV1alpha1WatchPathConfig**](IoArgoprojEventsV1alpha1WatchPathConfig.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_sftp_event_source import IoArgoprojEventsV1alpha1SFTPEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1SFTPEventSource from a JSON string
io_argoproj_events_v1alpha1_sftp_event_source_instance = IoArgoprojEventsV1alpha1SFTPEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1SFTPEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_sftp_event_source_dict = io_argoproj_events_v1alpha1_sftp_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1SFTPEventSource from a dict
io_argoproj_events_v1alpha1_sftp_event_source_form_dict = io_argoproj_events_v1alpha1_sftp_event_source.from_dict(io_argoproj_events_v1alpha1_sftp_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


