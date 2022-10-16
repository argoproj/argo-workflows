

# IoArgoprojEventsV1alpha1PulsarEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**authTokenSecret** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]
**connectionBackoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  |  [optional]
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**jsonBody** | **Boolean** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  |  [optional]
**tlsAllowInsecureConnection** | **Boolean** |  |  [optional]
**tlsTrustCertsSecret** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]
**tlsValidateHostname** | **Boolean** |  |  [optional]
**topics** | **List&lt;String&gt;** |  |  [optional]
**type** | **String** |  |  [optional]
**url** | **String** |  |  [optional]



