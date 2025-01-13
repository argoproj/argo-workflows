

# IoArgoprojEventsV1alpha1RedisStreamEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**consumerGroup** | **String** |  |  [optional]
**db** | **Integer** |  |  [optional]
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**hostAddress** | **String** |  |  [optional]
**maxMsgCountPerRead** | **Integer** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**password** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**streams** | **List&lt;String&gt;** | Streams to look for entries. XREADGROUP is used on all streams using a single consumer group. |  [optional]
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  |  [optional]
**username** | **String** |  |  [optional]



