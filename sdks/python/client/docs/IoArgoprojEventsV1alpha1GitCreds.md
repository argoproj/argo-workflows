# IoArgoprojEventsV1alpha1GitCreds


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**password** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**username** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_git_creds import IoArgoprojEventsV1alpha1GitCreds

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1GitCreds from a JSON string
io_argoproj_events_v1alpha1_git_creds_instance = IoArgoprojEventsV1alpha1GitCreds.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1GitCreds.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_git_creds_dict = io_argoproj_events_v1alpha1_git_creds_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1GitCreds from a dict
io_argoproj_events_v1alpha1_git_creds_form_dict = io_argoproj_events_v1alpha1_git_creds.from_dict(io_argoproj_events_v1alpha1_git_creds_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


