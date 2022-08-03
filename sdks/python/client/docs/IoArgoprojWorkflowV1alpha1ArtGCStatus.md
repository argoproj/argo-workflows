# IoArgoprojWorkflowV1alpha1ArtGCStatus

map ArtifactGC Pod name to phase

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**not_specified** | **bool** | if this is true, we already checked to see if we need to do it and we don&#39;t | [optional] 
**pods_recouped** | **{str: (bool,)}** | have completed Pods been processed? (mapped by Pod name) | [optional] 
**strategies_processed** | **{str: (bool,)}** | have Pods been started to perform this strategy? | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


