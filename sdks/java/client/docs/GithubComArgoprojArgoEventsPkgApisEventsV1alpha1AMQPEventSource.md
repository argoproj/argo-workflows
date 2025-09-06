

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BasicAuth.md) |  |  [optional]
**connectionBackoff** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff.md) |  |  [optional]
**consume** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPConsumeConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPConsumeConfig.md) |  |  [optional]
**exchangeDeclare** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPExchangeDeclareConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPExchangeDeclareConfig.md) |  |  [optional]
**exchangeName** | **String** |  |  [optional]
**exchangeType** | **String** |  |  [optional]
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**jsonBody** | **Boolean** |  |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**queueBind** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPQueueBindConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPQueueBindConfig.md) |  |  [optional]
**queueDeclare** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPQueueDeclareConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPQueueDeclareConfig.md) |  |  [optional]
**routingKey** | **String** |  |  [optional]
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  |  [optional]
**url** | **String** |  |  [optional]
**urlSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]



