

# IoArgoprojEventsV1alpha1GenericEventSource

GenericEventSource refers to a generic event source. It can be used to implement a custom event source.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**authSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**config** | **String** |  |  [optional]
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**insecure** | **Boolean** | Insecure determines the type of connection. |  [optional]
**jsonBody** | **Boolean** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**url** | **String** | URL of the gRPC server that implements the event source. |  [optional]



