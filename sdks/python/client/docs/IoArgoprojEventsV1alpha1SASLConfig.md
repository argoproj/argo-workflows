# IoArgoprojEventsV1alpha1SASLConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**mechanism** | **str** |  | [optional] 
**password_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**user_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_sasl_config import IoArgoprojEventsV1alpha1SASLConfig

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1SASLConfig from a JSON string
io_argoproj_events_v1alpha1_sasl_config_instance = IoArgoprojEventsV1alpha1SASLConfig.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1SASLConfig.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_sasl_config_dict = io_argoproj_events_v1alpha1_sasl_config_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1SASLConfig from a dict
io_argoproj_events_v1alpha1_sasl_config_form_dict = io_argoproj_events_v1alpha1_sasl_config.from_dict(io_argoproj_events_v1alpha1_sasl_config_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


