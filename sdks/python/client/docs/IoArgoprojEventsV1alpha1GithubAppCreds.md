# IoArgoprojEventsV1alpha1GithubAppCreds


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**app_id** | **str** |  | [optional] 
**installation_id** | **str** |  | [optional] 
**private_key** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_github_app_creds import IoArgoprojEventsV1alpha1GithubAppCreds

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1GithubAppCreds from a JSON string
io_argoproj_events_v1alpha1_github_app_creds_instance = IoArgoprojEventsV1alpha1GithubAppCreds.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1GithubAppCreds.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_github_app_creds_dict = io_argoproj_events_v1alpha1_github_app_creds_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1GithubAppCreds from a dict
io_argoproj_events_v1alpha1_github_app_creds_form_dict = io_argoproj_events_v1alpha1_github_app_creds.from_dict(io_argoproj_events_v1alpha1_github_app_creds_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


