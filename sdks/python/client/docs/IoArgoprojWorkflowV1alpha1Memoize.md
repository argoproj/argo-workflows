# IoArgoprojWorkflowV1alpha1Memoize

Memoization enables caching for the Outputs of the template

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cache** | [**IoArgoprojWorkflowV1alpha1Cache**](IoArgoprojWorkflowV1alpha1Cache.md) |  | 
**key** | **str** | Key is the key to use as the caching key | 
**max_age** | **str** | MaxAge is the maximum age (e.g. \&quot;180s\&quot;, \&quot;24h\&quot;) of an entry that is still considered valid. If an entry is older than the MaxAge, it will be ignored. | 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


