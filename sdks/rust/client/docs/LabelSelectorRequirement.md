# LabelSelectorRequirement

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **String** | key is the label key that the selector applies to. | 
**operator** | **String** | operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist. | 
**values** | Option<**Vec<String>**> | values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


