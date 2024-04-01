# IoArgoprojEventsV1alpha1WatchPathConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**directory** | **str** |  | [optional] 
**path** | **str** |  | [optional] 
**path_regexp** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_watch_path_config import IoArgoprojEventsV1alpha1WatchPathConfig

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1WatchPathConfig from a JSON string
io_argoproj_events_v1alpha1_watch_path_config_instance = IoArgoprojEventsV1alpha1WatchPathConfig.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1WatchPathConfig.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_watch_path_config_dict = io_argoproj_events_v1alpha1_watch_path_config_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1WatchPathConfig from a dict
io_argoproj_events_v1alpha1_watch_path_config_form_dict = io_argoproj_events_v1alpha1_watch_path_config.from_dict(io_argoproj_events_v1alpha1_watch_path_config_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


