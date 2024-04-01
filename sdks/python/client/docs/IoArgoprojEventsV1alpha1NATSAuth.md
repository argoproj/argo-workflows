# IoArgoprojEventsV1alpha1NATSAuth


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**basic** | [**IoArgoprojEventsV1alpha1BasicAuth**](IoArgoprojEventsV1alpha1BasicAuth.md) |  | [optional] 
**credential** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**nkey** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_nats_auth import IoArgoprojEventsV1alpha1NATSAuth

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1NATSAuth from a JSON string
io_argoproj_events_v1alpha1_nats_auth_instance = IoArgoprojEventsV1alpha1NATSAuth.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1NATSAuth.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_nats_auth_dict = io_argoproj_events_v1alpha1_nats_auth_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1NATSAuth from a dict
io_argoproj_events_v1alpha1_nats_auth_form_dict = io_argoproj_events_v1alpha1_nats_auth.from_dict(io_argoproj_events_v1alpha1_nats_auth_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


