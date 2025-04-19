# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaEventSource


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**config** | **str** | Yaml format Sarama config for Kafka connection. It follows the struct of sarama.Config. See https://github.com/IBM/sarama/blob/main/config.go e.g.  consumer:   fetch:     min: 1 net:   MaxOpenRequests: 5  +optional | [optional] 
**connection_backoff** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff.md) |  | [optional] 
**consumer_group** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaConsumerGroup**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaConsumerGroup.md) |  | [optional] 
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**limit_events_per_second** | **str** |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**partition** | **str** |  | [optional] 
**sasl** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SASLConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SASLConfig.md) |  | [optional] 
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  | [optional] 
**topic** | **str** |  | [optional] 
**url** | **str** |  | [optional] 
**version** | **str** |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


