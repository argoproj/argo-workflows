# NodeSelectorRequirement

A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **str** | The label key that the selector applies to. | 
**operator** | **str** | Represents a key&#39;s relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt. | 
**values** | **[str]** | An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


