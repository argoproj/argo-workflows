# IoArgoprojEventsV1alpha1ValueFromSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config_map_key_ref** | [**ConfigMapKeySelector**](ConfigMapKeySelector.md) |  | [optional] 
**secret_key_ref** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_value_from_source import IoArgoprojEventsV1alpha1ValueFromSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1ValueFromSource from a JSON string
io_argoproj_events_v1alpha1_value_from_source_instance = IoArgoprojEventsV1alpha1ValueFromSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1ValueFromSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_value_from_source_dict = io_argoproj_events_v1alpha1_value_from_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1ValueFromSource from a dict
io_argoproj_events_v1alpha1_value_from_source_form_dict = io_argoproj_events_v1alpha1_value_from_source.from_dict(io_argoproj_events_v1alpha1_value_from_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


