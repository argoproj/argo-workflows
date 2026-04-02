

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config** | **String** | Yaml format Sarama config for Kafka connection. It follows the struct of sarama.Config. See https://github.com/IBM/sarama/blob/main/config.go e.g.  consumer:   fetch:     min: 1 net:   MaxOpenRequests: 5  +optional |  [optional]
**connectionBackoff** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff.md) |  |  [optional]
**consumerGroup** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaConsumerGroup**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaConsumerGroup.md) |  |  [optional]
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**jsonBody** | **Boolean** |  |  [optional]
**limitEventsPerSecond** | **String** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**partition** | **String** |  |  [optional]
**sasl** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SASLConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SASLConfig.md) |  |  [optional]
**schemaRegistry** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SchemaRegistryConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SchemaRegistryConfig.md) |  |  [optional]
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  |  [optional]
**topic** | **String** |  |  [optional]
**url** | **String** |  |  [optional]
**version** | **String** |  |  [optional]



