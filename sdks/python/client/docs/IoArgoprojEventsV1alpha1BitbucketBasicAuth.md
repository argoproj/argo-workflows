# IoArgoprojEventsV1alpha1BitbucketBasicAuth


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**password** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**username** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_bitbucket_basic_auth import IoArgoprojEventsV1alpha1BitbucketBasicAuth

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1BitbucketBasicAuth from a JSON string
io_argoproj_events_v1alpha1_bitbucket_basic_auth_instance = IoArgoprojEventsV1alpha1BitbucketBasicAuth.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1BitbucketBasicAuth.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_bitbucket_basic_auth_dict = io_argoproj_events_v1alpha1_bitbucket_basic_auth_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1BitbucketBasicAuth from a dict
io_argoproj_events_v1alpha1_bitbucket_basic_auth_form_dict = io_argoproj_events_v1alpha1_bitbucket_basic_auth.from_dict(io_argoproj_events_v1alpha1_bitbucket_basic_auth_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


