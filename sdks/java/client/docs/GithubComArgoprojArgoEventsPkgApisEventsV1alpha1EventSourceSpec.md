

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**amqp** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPEventSource.md) |  |  [optional]
**azureEventsHub** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureEventsHubEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureEventsHubEventSource.md) |  |  [optional]
**azureQueueStorage** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureQueueStorageEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureQueueStorageEventSource.md) |  |  [optional]
**azureServiceBus** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureServiceBusEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureServiceBusEventSource.md) |  |  [optional]
**bitbucket** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketEventSource.md) |  |  [optional]
**bitbucketserver** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketServerEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketServerEventSource.md) |  |  [optional]
**calendar** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1CalendarEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1CalendarEventSource.md) |  |  [optional]
**emitter** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EmitterEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EmitterEventSource.md) |  |  [optional]
**eventBusName** | **String** |  |  [optional]
**file** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1FileEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1FileEventSource.md) |  |  [optional]
**generic** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GenericEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GenericEventSource.md) |  |  [optional]
**gerrit** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GerritEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GerritEventSource.md) |  |  [optional]
**github** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GithubEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GithubEventSource.md) |  |  [optional]
**gitlab** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitlabEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitlabEventSource.md) |  |  [optional]
**hdfs** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1HDFSEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1HDFSEventSource.md) |  |  [optional]
**kafka** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaEventSource.md) |  |  [optional]
**minio** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Artifact&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Artifact.md) |  |  [optional]
**mqtt** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1MQTTEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1MQTTEventSource.md) |  |  [optional]
**nats** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSEventsSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSEventsSource.md) |  |  [optional]
**nsq** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NSQEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NSQEventSource.md) |  |  [optional]
**pubSub** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PubSubEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PubSubEventSource.md) |  |  [optional]
**pulsar** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PulsarEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PulsarEventSource.md) |  |  [optional]
**redis** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RedisEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RedisEventSource.md) |  |  [optional]
**redisStream** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RedisStreamEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RedisStreamEventSource.md) |  |  [optional]
**replicas** | **Integer** |  |  [optional]
**resource** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ResourceEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ResourceEventSource.md) |  |  [optional]
**service** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Service**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Service.md) |  |  [optional]
**sftp** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SFTPEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SFTPEventSource.md) |  |  [optional]
**slack** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackEventSource.md) |  |  [optional]
**sns** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SNSEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SNSEventSource.md) |  |  [optional]
**sqs** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SQSEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SQSEventSource.md) |  |  [optional]
**storageGrid** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StorageGridEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StorageGridEventSource.md) |  |  [optional]
**stripe** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StripeEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StripeEventSource.md) |  |  [optional]
**template** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Template**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Template.md) |  |  [optional]
**webhook** | [**Map&lt;String, GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WebhookEventSource&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WebhookEventSource.md) |  |  [optional]



