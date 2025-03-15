# IoArgoprojWorkflowV1alpha1Backoff

Backoff is a backoff strategy to use within retryStrategy

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cap** | **str** | Cap is a limit on revised values of the duration parameter. If a multiplication by the factor parameter would make the duration exceed the cap then the duration is set to the cap | [optional] 
**duration** | **str** | Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. \&quot;2m\&quot;, \&quot;1h\&quot;) | [optional] 
**factor** | **str** |  | [optional] 
**max_duration** | **str** | MaxDuration is the maximum amount of time allowed for a workflow in the backoff strategy. It is important to note that if the workflow template includes activeDeadlineSeconds, the pod&#39;s deadline is initially set with activeDeadlineSeconds. However, when the workflow fails, the pod&#39;s deadline is then overridden by maxDuration. This ensures that the workflow does not exceed the specified maximum duration when retries are involved. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


