

# IoArgoprojEventsV1alpha1SQSEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**accessKey** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**dlq** | **Boolean** |  |  [optional]
**endpoint** | **String** |  |  [optional]
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**jsonBody** | **Boolean** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**queue** | **String** |  |  [optional]
**queueAccountId** | **String** |  |  [optional]
**region** | **String** |  |  [optional]
**roleARN** | **String** |  |  [optional]
**secretKey** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**sessionToken** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**waitTimeSeconds** | **String** | WaitTimeSeconds is The duration (in seconds) for which the call waits for a message to arrive in the queue before returning. |  [optional]



