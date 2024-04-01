# IoArgoprojEventsV1alpha1GitRemoteConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **str** | Name of the remote to fetch from. | [optional] 
**urls** | **List[str]** | URLs the URLs of a remote repository. It must be non-empty. Fetch will always use the first URL, while push will use all of them. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_git_remote_config import IoArgoprojEventsV1alpha1GitRemoteConfig

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1GitRemoteConfig from a JSON string
io_argoproj_events_v1alpha1_git_remote_config_instance = IoArgoprojEventsV1alpha1GitRemoteConfig.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1GitRemoteConfig.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_git_remote_config_dict = io_argoproj_events_v1alpha1_git_remote_config_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1GitRemoteConfig from a dict
io_argoproj_events_v1alpha1_git_remote_config_form_dict = io_argoproj_events_v1alpha1_git_remote_config.from_dict(io_argoproj_events_v1alpha1_git_remote_config_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


