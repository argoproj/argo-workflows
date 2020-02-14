# V1GetOptions

GetOptions is the standard query options to the standard REST get call.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**resource_version** | **str** | When specified: - if unset, then the result is returned from remote storage based on quorum-read flag; - if it&#39;s 0, then we simply return what we currently have in cache, no guarantee; - if set to non zero, then the result is at least as fresh as given rv. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


