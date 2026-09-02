

# IoArgoprojWorkflowV1alpha1RetryStrategy

RetryStrategy provides controls on how to retry a workflow step

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**affinity** | [**IoArgoprojWorkflowV1alpha1RetryAffinity**](IoArgoprojWorkflowV1alpha1RetryAffinity.md) |  |  [optional]
**backoff** | [**IoArgoprojWorkflowV1alpha1Backoff**](IoArgoprojWorkflowV1alpha1Backoff.md) |  |  [optional]
**expression** | **String** | Expression is a condition expression for when a node will be retried. If it evaluates to false, the node will not be retried and the retry strategy will be ignored |  [optional]
**limit** | **String** |  |  [optional]
**maxExecutionDuration** | **String** | MaxExecutionDuration is the maximum cumulative execution time of completed, pod-backed retry attempts. For each attempt, execution time spans the earliest observed main-container start through the latest main-container finish. If timestamps do not establish the latest main-container finish, execution time conservatively extends until Argo observes the attempt complete. Pending time, init containers, output processing, and retry backoff are otherwise excluded. The limit is checked only after a failed or errored attempt and never terminates an active or successful attempt. Parameterized values are resolved and captured when the retry sequence starts. |  [optional]
**retryPolicy** | **String** | RetryPolicy is a policy of NodePhase statuses that will be retried |  [optional]



