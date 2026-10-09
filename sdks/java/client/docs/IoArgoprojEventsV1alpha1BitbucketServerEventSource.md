

# IoArgoprojEventsV1alpha1BitbucketServerEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**accessToken** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**bitbucketserverBaseURL** | **String** | BitbucketServerBaseURL is the base URL for API requests to a custom endpoint. |  [optional]
**checkInterval** | **String** |  |  [optional]
**deleteHookOnFinish** | **Boolean** |  |  [optional]
**events** | **List&lt;String&gt;** |  |  [optional]
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**oneEventPerChange** | **Boolean** |  |  [optional]
**projectKey** | **String** |  |  [optional]
**projects** | **List&lt;String&gt;** |  |  [optional]
**repositories** | [**List&lt;IoArgoprojEventsV1alpha1BitbucketServerRepository&gt;**](IoArgoprojEventsV1alpha1BitbucketServerRepository.md) |  |  [optional]
**repositorySlug** | **String** |  |  [optional]
**skipBranchRefsChangedOnOpenPR** | **Boolean** |  |  [optional]
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  |  [optional]
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  |  [optional]
**webhookSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]



