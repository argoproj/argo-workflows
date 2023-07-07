# IoArgoprojWorkflowV1alpha1MutexStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**holding** | Option<[**Vec<crate::models::IoArgoprojWorkflowV1alpha1MutexHolding>**](io.argoproj.workflow.v1alpha1.MutexHolding.md)> | Holding is a list of mutexes and their respective objects that are held by mutex lock for this io.argoproj.workflow.v1alpha1. | [optional]
**waiting** | Option<[**Vec<crate::models::IoArgoprojWorkflowV1alpha1MutexHolding>**](io.argoproj.workflow.v1alpha1.MutexHolding.md)> | Waiting is a list of mutexes and their respective objects this workflow is waiting for. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


