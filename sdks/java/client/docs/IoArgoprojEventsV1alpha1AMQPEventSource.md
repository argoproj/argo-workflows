

# IoArgoprojEventsV1alpha1AMQPEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**IoArgoprojEventsV1alpha1BasicAuth**](IoArgoprojEventsV1alpha1BasicAuth.md) |  |  [optional]
**connectionBackoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  |  [optional]
**consume** | [**IoArgoprojEventsV1alpha1AMQPConsumeConfig**](IoArgoprojEventsV1alpha1AMQPConsumeConfig.md) |  |  [optional]
**exchangeDeclare** | [**IoArgoprojEventsV1alpha1AMQPExchangeDeclareConfig**](IoArgoprojEventsV1alpha1AMQPExchangeDeclareConfig.md) |  |  [optional]
**exchangeName** | **String** |  |  [optional]
**exchangeType** | **String** |  |  [optional]
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**jsonBody** | **Boolean** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**queueBind** | [**IoArgoprojEventsV1alpha1AMQPQueueBindConfig**](IoArgoprojEventsV1alpha1AMQPQueueBindConfig.md) |  |  [optional]
**queueDeclare** | [**IoArgoprojEventsV1alpha1AMQPQueueDeclareConfig**](IoArgoprojEventsV1alpha1AMQPQueueDeclareConfig.md) |  |  [optional]
**routingKey** | **String** |  |  [optional]
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  |  [optional]
**url** | **String** |  |  [optional]
**urlSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]



