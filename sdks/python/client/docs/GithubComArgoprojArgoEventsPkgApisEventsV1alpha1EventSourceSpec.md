# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceSpec


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**amqp** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AMQPEventSource.md) |  | [optional] 
**azure_events_hub** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureEventsHubEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureEventsHubEventSource.md) |  | [optional] 
**azure_queue_storage** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureQueueStorageEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureQueueStorageEventSource.md) |  | [optional] 
**azure_service_bus** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureServiceBusEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureServiceBusEventSource.md) |  | [optional] 
**bitbucket** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketEventSource.md) |  | [optional] 
**bitbucketserver** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketServerEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1BitbucketServerEventSource.md) |  | [optional] 
**calendar** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1CalendarEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1CalendarEventSource.md) |  | [optional] 
**emitter** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EmitterEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EmitterEventSource.md) |  | [optional] 
**event_bus_name** | **str** |  | [optional] 
**file** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1FileEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1FileEventSource.md) |  | [optional] 
**generic** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GenericEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GenericEventSource.md) |  | [optional] 
**gerrit** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GerritEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GerritEventSource.md) |  | [optional] 
**github** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GithubEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GithubEventSource.md) |  | [optional] 
**gitlab** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitlabEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1GitlabEventSource.md) |  | [optional] 
**hdfs** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1HDFSEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1HDFSEventSource.md) |  | [optional] 
**kafka** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaEventSource.md) |  | [optional] 
**minio** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Artifact,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1S3Artifact.md) |  | [optional] 
**mqtt** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1MQTTEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1MQTTEventSource.md) |  | [optional] 
**nats** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSEventsSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSEventsSource.md) |  | [optional] 
**nsq** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NSQEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NSQEventSource.md) |  | [optional] 
**pub_sub** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PubSubEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PubSubEventSource.md) |  | [optional] 
**pulsar** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PulsarEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PulsarEventSource.md) |  | [optional] 
**redis** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RedisEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RedisEventSource.md) |  | [optional] 
**redis_stream** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RedisStreamEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1RedisStreamEventSource.md) |  | [optional] 
**replicas** | **int** |  | [optional] 
**resource** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ResourceEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ResourceEventSource.md) |  | [optional] 
**service** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Service**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Service.md) |  | [optional] 
**sftp** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SFTPEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SFTPEventSource.md) |  | [optional] 
**slack** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackEventSource.md) |  | [optional] 
**sns** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SNSEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SNSEventSource.md) |  | [optional] 
**sqs** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SQSEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SQSEventSource.md) |  | [optional] 
**storage_grid** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StorageGridEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StorageGridEventSource.md) |  | [optional] 
**stripe** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StripeEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StripeEventSource.md) |  | [optional] 
**template** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Template**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Template.md) |  | [optional] 
**webhook** | [**{str: (GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WebhookEventSource,)}**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WebhookEventSource.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


