# IoArgoprojEventsV1alpha1EventSourceSpec


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**amqp** | [**{str: (IoArgoprojEventsV1alpha1AMQPEventSource,)}**](IoArgoprojEventsV1alpha1AMQPEventSource.md) |  | [optional] 
**azure_events_hub** | [**{str: (IoArgoprojEventsV1alpha1AzureEventsHubEventSource,)}**](IoArgoprojEventsV1alpha1AzureEventsHubEventSource.md) |  | [optional] 
**azure_queue_storage** | [**{str: (IoArgoprojEventsV1alpha1AzureQueueStorageEventSource,)}**](IoArgoprojEventsV1alpha1AzureQueueStorageEventSource.md) |  | [optional] 
**azure_service_bus** | [**{str: (IoArgoprojEventsV1alpha1AzureServiceBusEventSource,)}**](IoArgoprojEventsV1alpha1AzureServiceBusEventSource.md) |  | [optional] 
**bitbucket** | [**{str: (IoArgoprojEventsV1alpha1BitbucketEventSource,)}**](IoArgoprojEventsV1alpha1BitbucketEventSource.md) |  | [optional] 
**bitbucketserver** | [**{str: (IoArgoprojEventsV1alpha1BitbucketServerEventSource,)}**](IoArgoprojEventsV1alpha1BitbucketServerEventSource.md) |  | [optional] 
**calendar** | [**{str: (IoArgoprojEventsV1alpha1CalendarEventSource,)}**](IoArgoprojEventsV1alpha1CalendarEventSource.md) |  | [optional] 
**emitter** | [**{str: (IoArgoprojEventsV1alpha1EmitterEventSource,)}**](IoArgoprojEventsV1alpha1EmitterEventSource.md) |  | [optional] 
**event_bus_name** | **str** |  | [optional] 
**file** | [**{str: (IoArgoprojEventsV1alpha1FileEventSource,)}**](IoArgoprojEventsV1alpha1FileEventSource.md) |  | [optional] 
**generic** | [**{str: (IoArgoprojEventsV1alpha1GenericEventSource,)}**](IoArgoprojEventsV1alpha1GenericEventSource.md) |  | [optional] 
**gerrit** | [**{str: (IoArgoprojEventsV1alpha1GerritEventSource,)}**](IoArgoprojEventsV1alpha1GerritEventSource.md) |  | [optional] 
**github** | [**{str: (IoArgoprojEventsV1alpha1GithubEventSource,)}**](IoArgoprojEventsV1alpha1GithubEventSource.md) |  | [optional] 
**gitlab** | [**{str: (IoArgoprojEventsV1alpha1GitlabEventSource,)}**](IoArgoprojEventsV1alpha1GitlabEventSource.md) |  | [optional] 
**hdfs** | [**{str: (IoArgoprojEventsV1alpha1HDFSEventSource,)}**](IoArgoprojEventsV1alpha1HDFSEventSource.md) |  | [optional] 
**kafka** | [**{str: (IoArgoprojEventsV1alpha1KafkaEventSource,)}**](IoArgoprojEventsV1alpha1KafkaEventSource.md) |  | [optional] 
**minio** | [**{str: (IoArgoprojEventsV1alpha1S3Artifact,)}**](IoArgoprojEventsV1alpha1S3Artifact.md) |  | [optional] 
**mqtt** | [**{str: (IoArgoprojEventsV1alpha1MQTTEventSource,)}**](IoArgoprojEventsV1alpha1MQTTEventSource.md) |  | [optional] 
**nats** | [**{str: (IoArgoprojEventsV1alpha1NATSEventsSource,)}**](IoArgoprojEventsV1alpha1NATSEventsSource.md) |  | [optional] 
**nsq** | [**{str: (IoArgoprojEventsV1alpha1NSQEventSource,)}**](IoArgoprojEventsV1alpha1NSQEventSource.md) |  | [optional] 
**pub_sub** | [**{str: (IoArgoprojEventsV1alpha1PubSubEventSource,)}**](IoArgoprojEventsV1alpha1PubSubEventSource.md) |  | [optional] 
**pulsar** | [**{str: (IoArgoprojEventsV1alpha1PulsarEventSource,)}**](IoArgoprojEventsV1alpha1PulsarEventSource.md) |  | [optional] 
**redis** | [**{str: (IoArgoprojEventsV1alpha1RedisEventSource,)}**](IoArgoprojEventsV1alpha1RedisEventSource.md) |  | [optional] 
**redis_stream** | [**{str: (IoArgoprojEventsV1alpha1RedisStreamEventSource,)}**](IoArgoprojEventsV1alpha1RedisStreamEventSource.md) |  | [optional] 
**replicas** | **int** |  | [optional] 
**resource** | [**{str: (IoArgoprojEventsV1alpha1ResourceEventSource,)}**](IoArgoprojEventsV1alpha1ResourceEventSource.md) |  | [optional] 
**service** | [**IoArgoprojEventsV1alpha1Service**](IoArgoprojEventsV1alpha1Service.md) |  | [optional] 
**sftp** | [**{str: (IoArgoprojEventsV1alpha1SFTPEventSource,)}**](IoArgoprojEventsV1alpha1SFTPEventSource.md) |  | [optional] 
**slack** | [**{str: (IoArgoprojEventsV1alpha1SlackEventSource,)}**](IoArgoprojEventsV1alpha1SlackEventSource.md) |  | [optional] 
**sns** | [**{str: (IoArgoprojEventsV1alpha1SNSEventSource,)}**](IoArgoprojEventsV1alpha1SNSEventSource.md) |  | [optional] 
**sqs** | [**{str: (IoArgoprojEventsV1alpha1SQSEventSource,)}**](IoArgoprojEventsV1alpha1SQSEventSource.md) |  | [optional] 
**storage_grid** | [**{str: (IoArgoprojEventsV1alpha1StorageGridEventSource,)}**](IoArgoprojEventsV1alpha1StorageGridEventSource.md) |  | [optional] 
**stripe** | [**{str: (IoArgoprojEventsV1alpha1StripeEventSource,)}**](IoArgoprojEventsV1alpha1StripeEventSource.md) |  | [optional] 
**template** | [**IoArgoprojEventsV1alpha1Template**](IoArgoprojEventsV1alpha1Template.md) |  | [optional] 
**webhook** | [**{str: (IoArgoprojEventsV1alpha1WebhookEventSource,)}**](IoArgoprojEventsV1alpha1WebhookEventSource.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


