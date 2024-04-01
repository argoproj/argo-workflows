# IoArgoprojEventsV1alpha1TLSConfig

TLSConfig refers to TLS configuration for a client.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ca_cert_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**client_cert_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**client_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**insecure_skip_verify** | **bool** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_tls_config import IoArgoprojEventsV1alpha1TLSConfig

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1TLSConfig from a JSON string
io_argoproj_events_v1alpha1_tls_config_instance = IoArgoprojEventsV1alpha1TLSConfig.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1TLSConfig.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_tls_config_dict = io_argoproj_events_v1alpha1_tls_config_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1TLSConfig from a dict
io_argoproj_events_v1alpha1_tls_config_form_dict = io_argoproj_events_v1alpha1_tls_config.from_dict(io_argoproj_events_v1alpha1_tls_config_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


