

# IoArgoprojEventsV1alpha1GithubEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active** | **Boolean** |  |  [optional]
**apiToken** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**contentType** | **String** |  |  [optional]
**deleteHookOnFinish** | **Boolean** |  |  [optional]
**events** | **List&lt;String&gt;** |  |  [optional]
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**githubApp** | [**IoArgoprojEventsV1alpha1GithubAppCreds**](IoArgoprojEventsV1alpha1GithubAppCreds.md) |  |  [optional]
**githubBaseURL** | **String** |  |  [optional]
**githubUploadURL** | **String** |  |  [optional]
**id** | **String** |  |  [optional]
**insecure** | **Boolean** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**organizations** | **List&lt;String&gt;** | Organizations holds the names of organizations (used for organization level webhooks). Not required if Repositories is set. |  [optional]
**owner** | **String** |  |  [optional]
**repositories** | [**List&lt;IoArgoprojEventsV1alpha1OwnedRepositories&gt;**](IoArgoprojEventsV1alpha1OwnedRepositories.md) | Repositories holds the information of repositories, which uses repo owner as the key, and list of repo names as the value. Not required if Organizations is set. |  [optional]
**repository** | **String** |  |  [optional]
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  |  [optional]
**webhookSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]



