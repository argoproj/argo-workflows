# StatusCause

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**field** | Option<**String**> | The field of the resource that has caused this error, as named by its JSON serialization. May include dot and postfix notation for nested attributes. Arrays are zero-indexed.  Fields may appear more than once in an array of causes due to fields having multiple errors. Optional.  Examples:   \"name\" - the field \"name\" on the current resource   \"items[0].name\" - the field \"name\" on the first array entry in \"items\" | [optional]
**message** | Option<**String**> | A human-readable description of the cause of the error.  This field may be presented as-is to a reader. | [optional]
**reason** | Option<**String**> | A machine-readable description of the cause of the error. If this value is empty there is no information available. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


