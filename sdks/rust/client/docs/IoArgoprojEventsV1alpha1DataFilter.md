# IoArgoprojEventsV1alpha1DataFilter

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**comparator** | Option<**String**> | Comparator compares the event data with a user given value. Can be \">=\", \">\", \"=\", \"!=\", \"<\", or \"<=\". Is optional, and if left blank treated as equality \"=\". | [optional]
**path** | Option<**String**> | Path is the JSONPath of the event's (JSON decoded) data key Path is a series of keys separated by a dot. A key may contain wildcard characters '*' and '?'. To access an array value use the index as the key. The dot and wildcard characters can be escaped with '\\\\'. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. | [optional]
**template** | Option<**String**> |  | [optional]
**_type** | Option<**String**> |  | [optional]
**value** | Option<**Vec<String>**> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


