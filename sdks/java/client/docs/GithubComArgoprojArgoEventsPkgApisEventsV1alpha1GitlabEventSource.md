

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitlabEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**accessToken** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**deleteHookOnFinish** | **Boolean** |  |  [optional]
**enableSSLVerification** | **Boolean** |  |  [optional]
**events** | **List&lt;String&gt;** | Events are gitlab event to listen to. Refer https://github.com/xanzy/go-gitlab/blob/bf34eca5d13a9f4c3f501d8a97b8ac226d55e4d9/projects.go#L794. |  [optional]
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**gitlabBaseURL** | **String** |  |  [optional]
**groups** | **List&lt;String&gt;** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**projectID** | **String** |  |  [optional]
**projects** | **List&lt;String&gt;** |  |  [optional]
**secretToken** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**webhook** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WebhookContext**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WebhookContext.md) |  |  [optional]



