

# IoArgoprojEventsV1alpha1KafkaEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config** | **String** | Yaml format Sarama config for Kafka connection. It follows the struct of sarama.Config. See https://github.com/Shopify/sarama/blob/main/config.go e.g.  consumer:   fetch:     min: 1 net:   MaxOpenRequests: 5  +optional |  [optional]
**connectionBackoff** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  |  [optional]
**consumerGroup** | [**IoArgoprojEventsV1alpha1KafkaConsumerGroup**](IoArgoprojEventsV1alpha1KafkaConsumerGroup.md) |  |  [optional]
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**jsonBody** | **Boolean** |  |  [optional]
**limitEventsPerSecond** | **String** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**partition** | **String** |  |  [optional]
**sasl** | [**IoArgoprojEventsV1alpha1SASLConfig**](IoArgoprojEventsV1alpha1SASLConfig.md) |  |  [optional]
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  |  [optional]
**topic** | **String** |  |  [optional]
**url** | **String** |  |  [optional]
**version** | **String** |  |  [optional]



