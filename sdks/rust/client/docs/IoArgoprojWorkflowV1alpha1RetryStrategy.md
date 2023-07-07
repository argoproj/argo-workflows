# IoArgoprojWorkflowV1alpha1RetryStrategy

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1RetryAffinity**](io.argoproj.workflow.v1alpha1.RetryAffinity.md)> |  | [optional]
**backoff** | Option<[**crate::models::IoArgoprojWorkflowV1alpha1Backoff**](io.argoproj.workflow.v1alpha1.Backoff.md)> |  | [optional]
**expression** | Option<**String**> | Expression is a condition expression for when a node will be retried. If it evaluates to false, the node will not be retried and the retry strategy will be ignored | [optional]
**limit** | Option<**String**> |  | [optional]
**retry_policy** | Option<**String**> | RetryPolicy is a policy of NodePhase statuses that will be retried | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


