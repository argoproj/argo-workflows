# IoArgoprojEventsV1alpha1EventSourceSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**amqp** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1AmqpEventSource>**](io.argoproj.events.v1alpha1.AMQPEventSource.md)> |  | [optional]
**azure_events_hub** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1AzureEventsHubEventSource>**](io.argoproj.events.v1alpha1.AzureEventsHubEventSource.md)> |  | [optional]
**bitbucket** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1BitbucketEventSource>**](io.argoproj.events.v1alpha1.BitbucketEventSource.md)> |  | [optional]
**bitbucketserver** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1BitbucketServerEventSource>**](io.argoproj.events.v1alpha1.BitbucketServerEventSource.md)> |  | [optional]
**calendar** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1CalendarEventSource>**](io.argoproj.events.v1alpha1.CalendarEventSource.md)> |  | [optional]
**emitter** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1EmitterEventSource>**](io.argoproj.events.v1alpha1.EmitterEventSource.md)> |  | [optional]
**event_bus_name** | Option<**String**> |  | [optional]
**file** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1FileEventSource>**](io.argoproj.events.v1alpha1.FileEventSource.md)> |  | [optional]
**generic** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1GenericEventSource>**](io.argoproj.events.v1alpha1.GenericEventSource.md)> |  | [optional]
**github** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1GithubEventSource>**](io.argoproj.events.v1alpha1.GithubEventSource.md)> |  | [optional]
**gitlab** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1GitlabEventSource>**](io.argoproj.events.v1alpha1.GitlabEventSource.md)> |  | [optional]
**hdfs** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1HdfsEventSource>**](io.argoproj.events.v1alpha1.HDFSEventSource.md)> |  | [optional]
**kafka** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1KafkaEventSource>**](io.argoproj.events.v1alpha1.KafkaEventSource.md)> |  | [optional]
**minio** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1S3Artifact>**](io.argoproj.events.v1alpha1.S3Artifact.md)> |  | [optional]
**mqtt** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1MqttEventSource>**](io.argoproj.events.v1alpha1.MQTTEventSource.md)> |  | [optional]
**nats** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1NatsEventsSource>**](io.argoproj.events.v1alpha1.NATSEventsSource.md)> |  | [optional]
**nsq** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1NsqEventSource>**](io.argoproj.events.v1alpha1.NSQEventSource.md)> |  | [optional]
**pub_sub** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1PubSubEventSource>**](io.argoproj.events.v1alpha1.PubSubEventSource.md)> |  | [optional]
**pulsar** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1PulsarEventSource>**](io.argoproj.events.v1alpha1.PulsarEventSource.md)> |  | [optional]
**redis** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1RedisEventSource>**](io.argoproj.events.v1alpha1.RedisEventSource.md)> |  | [optional]
**redis_stream** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1RedisStreamEventSource>**](io.argoproj.events.v1alpha1.RedisStreamEventSource.md)> |  | [optional]
**replicas** | Option<**i32**> |  | [optional]
**resource** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1ResourceEventSource>**](io.argoproj.events.v1alpha1.ResourceEventSource.md)> |  | [optional]
**service** | Option<[**crate::models::IoArgoprojEventsV1alpha1Service**](io.argoproj.events.v1alpha1.Service.md)> |  | [optional]
**slack** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1SlackEventSource>**](io.argoproj.events.v1alpha1.SlackEventSource.md)> |  | [optional]
**sns** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1SnsEventSource>**](io.argoproj.events.v1alpha1.SNSEventSource.md)> |  | [optional]
**sqs** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1SqsEventSource>**](io.argoproj.events.v1alpha1.SQSEventSource.md)> |  | [optional]
**storage_grid** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1StorageGridEventSource>**](io.argoproj.events.v1alpha1.StorageGridEventSource.md)> |  | [optional]
**stripe** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1StripeEventSource>**](io.argoproj.events.v1alpha1.StripeEventSource.md)> |  | [optional]
**template** | Option<[**crate::models::IoArgoprojEventsV1alpha1Template**](io.argoproj.events.v1alpha1.Template.md)> |  | [optional]
**webhook** | Option<[**::std::collections::HashMap<String, crate::models::IoArgoprojEventsV1alpha1WebhookEventSource>**](io.argoproj.events.v1alpha1.WebhookEventSource.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


