# IoArgoprojEventsV1alpha1TriggerParameter

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**dest** | Option<**String**> | Dest is the JSONPath of a resource key. A path is a series of keys separated by a dot. The colon character can be escaped with '.' The -1 key can be used to append a value to an existing array. See https://github.com/tidwall/sjson#path-syntax for more information about how this is used. | [optional]
**operation** | Option<**String**> | Operation is what to do with the existing value at Dest, whether to 'prepend', 'overwrite', or 'append' it. | [optional]
**src** | Option<[**crate::models::IoArgoprojEventsV1alpha1TriggerParameterSource**](io.argoproj.events.v1alpha1.TriggerParameterSource.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


