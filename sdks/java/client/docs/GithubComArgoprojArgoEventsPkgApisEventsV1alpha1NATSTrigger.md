

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSTrigger

NATSTrigger refers to the specification of the NATS trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSAuth**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSAuth.md) |  |  [optional]
**parameters** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) |  |  [optional]
**payload** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameter.md) |  |  [optional]
**subject** | **String** | Name of the subject to put message on. |  [optional]
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  |  [optional]
**url** | **String** | URL of the NATS cluster. |  [optional]



