

# StatusCause

StatusCause provides more information about an api.Status failure, including cases when multiple errors are encountered.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**field** | **String** | The field of the resource that has caused this error, as named by its JSON serialization. May include dot and postfix notation for nested attributes. Arrays are zero-indexed.  Fields may appear more than once in an array of causes due to fields having multiple errors. Optional.  Examples:   \&quot;name\&quot; - the field \&quot;name\&quot; on the current resource   \&quot;items[0].name\&quot; - the field \&quot;name\&quot; on the first array entry in \&quot;items\&quot; |  [optional]
**message** | **String** | A human-readable description of the cause of the error.  This field may be presented as-is to a reader. |  [optional]
**reason** | **String** | A machine-readable description of the cause of the error. If this value is empty there is no information available. |  [optional]



