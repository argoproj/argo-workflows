# IoArgoprojEventsV1alpha1DataFilter


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**comparator** | **str** | Comparator compares the event data with a user given value. Can be \&quot;&gt;&#x3D;\&quot;, \&quot;&gt;\&quot;, \&quot;&#x3D;\&quot;, \&quot;!&#x3D;\&quot;, \&quot;&lt;\&quot;, or \&quot;&lt;&#x3D;\&quot;. Is optional, and if left blank treated as equality \&quot;&#x3D;\&quot;. | [optional] 
**path** | **str** | Path is the JSONPath of the event&#39;s (JSON decoded) data key Path is a series of keys separated by a dot. A key may contain wildcard characters &#39;*&#39; and &#39;?&#39;. To access an array value use the index as the key. The dot and wildcard characters can be escaped with &#39;\\\\&#39;. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. | [optional] 
**template** | **str** |  | [optional] 
**type** | **str** |  | [optional] 
**value** | **List[str]** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_data_filter import IoArgoprojEventsV1alpha1DataFilter

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1DataFilter from a JSON string
io_argoproj_events_v1alpha1_data_filter_instance = IoArgoprojEventsV1alpha1DataFilter.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1DataFilter.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_data_filter_dict = io_argoproj_events_v1alpha1_data_filter_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1DataFilter from a dict
io_argoproj_events_v1alpha1_data_filter_form_dict = io_argoproj_events_v1alpha1_data_filter.from_dict(io_argoproj_events_v1alpha1_data_filter_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


