

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureServiceBusEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**connectionString** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**deferDelete** | **Boolean** | DeferDelete controls when messages are removed from Azure Service Bus. If false (default), messages are received and deleted immediately before processing. If true, messages are locked and only deleted after successful processing, ensuring they are not lost if processing fails. |  [optional]
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**fullyQualifiedNamespace** | **String** |  |  [optional]
**jsonBody** | **Boolean** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**queueName** | **String** |  |  [optional]
**subscriptionName** | **String** |  |  [optional]
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  |  [optional]
**topicName** | **String** |  |  [optional]



