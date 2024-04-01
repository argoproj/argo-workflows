# IoArgoprojEventsV1alpha1GitArtifact


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**branch** | **str** |  | [optional] 
**clone_directory** | **str** | Directory to clone the repository. We clone complete directory because GitArtifact is not limited to any specific Git service providers. Hence we don&#39;t use any specific git provider client. | [optional] 
**creds** | [**IoArgoprojEventsV1alpha1GitCreds**](IoArgoprojEventsV1alpha1GitCreds.md) |  | [optional] 
**file_path** | **str** |  | [optional] 
**insecure_ignore_host_key** | **bool** |  | [optional] 
**ref** | **str** |  | [optional] 
**remote** | [**IoArgoprojEventsV1alpha1GitRemoteConfig**](IoArgoprojEventsV1alpha1GitRemoteConfig.md) |  | [optional] 
**ssh_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**tag** | **str** |  | [optional] 
**url** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_git_artifact import IoArgoprojEventsV1alpha1GitArtifact

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1GitArtifact from a JSON string
io_argoproj_events_v1alpha1_git_artifact_instance = IoArgoprojEventsV1alpha1GitArtifact.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1GitArtifact.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_git_artifact_dict = io_argoproj_events_v1alpha1_git_artifact_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1GitArtifact from a dict
io_argoproj_events_v1alpha1_git_artifact_form_dict = io_argoproj_events_v1alpha1_git_artifact.from_dict(io_argoproj_events_v1alpha1_git_artifact_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


