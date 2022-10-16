

# IoArgoprojEventsV1alpha1BitbucketServerEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**accessToken** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]
**bitbucketserverBaseURL** | **String** |  |  [optional]
**deleteHookOnFinish** | **Boolean** |  |  [optional]
**events** | **List&lt;String&gt;** |  |  [optional]
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**projectKey** | **String** |  |  [optional]
**repositories** | [**List&lt;IoArgoprojEventsV1alpha1BitbucketServerRepository&gt;**](IoArgoprojEventsV1alpha1BitbucketServerRepository.md) |  |  [optional]
**repositorySlug** | **String** |  |  [optional]
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  |  [optional]
**webhookSecret** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]



