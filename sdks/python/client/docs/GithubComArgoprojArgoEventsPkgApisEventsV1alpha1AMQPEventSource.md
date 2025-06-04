# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPEventSource


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth.md) |  | [optional] 
**connection_backoff** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff.md) |  | [optional] 
**consume** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPConsumeConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPConsumeConfig.md) |  | [optional] 
**exchange_declare** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPExchangeDeclareConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPExchangeDeclareConfig.md) |  | [optional] 
**exchange_name** | **str** |  | [optional] 
**exchange_type** | **str** |  | [optional] 
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**queue_bind** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPQueueBindConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPQueueBindConfig.md) |  | [optional] 
**queue_declare** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPQueueDeclareConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPQueueDeclareConfig.md) |  | [optional] 
**routing_key** | **str** |  | [optional] 
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  | [optional] 
**url** | **str** |  | [optional] 
**url_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


