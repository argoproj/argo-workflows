

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EmailTrigger

EmailTrigger refers to the specification of the email notification trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**body** | **String** |  |  [optional]
**from** | **String** |  |  [optional]
**host** | **String** | Host refers to the smtp host url to which email is send. |  [optional]
**parameters** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) |  |  [optional]
**port** | **Integer** |  |  [optional]
**smtpPassword** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**subject** | **String** |  |  [optional]
**to** | **List&lt;String&gt;** |  |  [optional]
**username** | **String** |  |  [optional]



