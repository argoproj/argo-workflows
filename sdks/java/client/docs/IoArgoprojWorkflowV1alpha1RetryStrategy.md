

# IoArgoprojWorkflowV1alpha1RetryStrategy

RetryStrategy provides controls on how to retry a workflow step

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | [**IoArgoprojWorkflowV1alpha1RetryAffinity**](IoArgoprojWorkflowV1alpha1RetryAffinity.md) |  |  [optional]
**backoff** | [**IoArgoprojWorkflowV1alpha1Backoff**](IoArgoprojWorkflowV1alpha1Backoff.md) |  |  [optional]
**expression** | **String** | Expression is a condition expression for when a node will be retried. If it evaluates to false, the node will not be retried and the retry strategy will be ignored |  [optional]
**limit** | **String** |  |  [optional]
**retryPolicy** | **String** | RetryPolicy is a policy of NodePhase statuses that will be retried |  [optional]



