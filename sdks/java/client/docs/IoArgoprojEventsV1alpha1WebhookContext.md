

# IoArgoprojEventsV1alpha1WebhookContext


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**authSecret** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]
**endpoint** | **String** |  |  [optional]
**maxPayloadSize** | **String** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**method** | **String** |  |  [optional]
**port** | **String** | Port on which HTTP server is listening for incoming events. |  [optional]
**serverCertSecret** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]
**serverKeySecret** | [**SecretKeySelector**](SecretKeySelector.md) |  |  [optional]
**url** | **String** | URL is the url of the server. |  [optional]



