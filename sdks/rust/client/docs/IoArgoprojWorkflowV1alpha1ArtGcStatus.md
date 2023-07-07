# IoArgoprojWorkflowV1alpha1ArtGcStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**not_specified** | Option<**bool**> | if this is true, we already checked to see if we need to do it and we don't | [optional]
**pods_recouped** | Option<**::std::collections::HashMap<String, bool>**> | have completed Pods been processed? (mapped by Pod name) used to prevent re-processing the Status of a Pod more than once | [optional]
**strategies_processed** | Option<**::std::collections::HashMap<String, bool>**> | have Pods been started to perform this strategy? (enables us not to re-process what we've already done) | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


