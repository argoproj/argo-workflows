# IoArgoprojEventsV1alpha1TriggerParameterSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**context_key** | **str** | ContextKey is the JSONPath of the event&#39;s (JSON decoded) context key ContextKey is a series of keys separated by a dot. A key may contain wildcard characters &#39;*&#39; and &#39;?&#39;. To access an array value use the index as the key. The dot and wildcard characters can be escaped with &#39;\\\\&#39;. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. | [optional] 
**context_template** | **str** |  | [optional] 
**data_key** | **str** | DataKey is the JSONPath of the event&#39;s (JSON decoded) data key DataKey is a series of keys separated by a dot. A key may contain wildcard characters &#39;*&#39; and &#39;?&#39;. To access an array value use the index as the key. The dot and wildcard characters can be escaped with &#39;\\\\&#39;. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. | [optional] 
**data_template** | **str** |  | [optional] 
**dependency_name** | **str** | DependencyName refers to the name of the dependency. The event which is stored for this dependency is used as payload for the parameterization. Make sure to refer to one of the dependencies you have defined under Dependencies list. | [optional] 
**use_raw_data** | **bool** |  | [optional] 
**value** | **str** | Value is the default literal value to use for this parameter source This is only used if the DataKey is invalid. If the DataKey is invalid and this is not defined, this param source will produce an error. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_trigger_parameter_source import IoArgoprojEventsV1alpha1TriggerParameterSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1TriggerParameterSource from a JSON string
io_argoproj_events_v1alpha1_trigger_parameter_source_instance = IoArgoprojEventsV1alpha1TriggerParameterSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1TriggerParameterSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_trigger_parameter_source_dict = io_argoproj_events_v1alpha1_trigger_parameter_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1TriggerParameterSource from a dict
io_argoproj_events_v1alpha1_trigger_parameter_source_form_dict = io_argoproj_events_v1alpha1_trigger_parameter_source.from_dict(io_argoproj_events_v1alpha1_trigger_parameter_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


