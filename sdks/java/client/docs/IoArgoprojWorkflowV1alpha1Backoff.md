

# IoArgoprojWorkflowV1alpha1Backoff

Backoff is a backoff strategy to use within retryStrategy

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cap** | **String** | Cap is a limit on revised values of the duration parameter. If a multiplication by the factor parameter would make the duration exceed the cap then the duration is set to the cap |  [optional]
**duration** | **String** | Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. \&quot;2m\&quot;, \&quot;1h\&quot;) |  [optional]
**factor** | **String** |  |  [optional]
**maxDuration** | **String** | MaxDuration is the maximum amount of time allowed for a workflow in the backoff strategy. It is important to note that if the workflow template includes activeDeadlineSeconds, the pod&#39;s deadline is initially set with activeDeadlineSeconds. However, when the workflow fails, the pod&#39;s deadline is then overridden by maxDuration. This ensures that the workflow does not exceed the specified maximum duration when retries are involved. |  [optional]



