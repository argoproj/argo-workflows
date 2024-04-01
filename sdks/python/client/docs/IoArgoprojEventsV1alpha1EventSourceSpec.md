# IoArgoprojEventsV1alpha1EventSourceSpec


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**amqp** | [**Dict[str, IoArgoprojEventsV1alpha1AMQPEventSource]**](IoArgoprojEventsV1alpha1AMQPEventSource.md) |  | [optional] 
**azure_events_hub** | [**Dict[str, IoArgoprojEventsV1alpha1AzureEventsHubEventSource]**](IoArgoprojEventsV1alpha1AzureEventsHubEventSource.md) |  | [optional] 
**azure_queue_storage** | [**Dict[str, IoArgoprojEventsV1alpha1AzureQueueStorageEventSource]**](IoArgoprojEventsV1alpha1AzureQueueStorageEventSource.md) |  | [optional] 
**azure_service_bus** | [**Dict[str, IoArgoprojEventsV1alpha1AzureServiceBusEventSource]**](IoArgoprojEventsV1alpha1AzureServiceBusEventSource.md) |  | [optional] 
**bitbucket** | [**Dict[str, IoArgoprojEventsV1alpha1BitbucketEventSource]**](IoArgoprojEventsV1alpha1BitbucketEventSource.md) |  | [optional] 
**bitbucketserver** | [**Dict[str, IoArgoprojEventsV1alpha1BitbucketServerEventSource]**](IoArgoprojEventsV1alpha1BitbucketServerEventSource.md) |  | [optional] 
**calendar** | [**Dict[str, IoArgoprojEventsV1alpha1CalendarEventSource]**](IoArgoprojEventsV1alpha1CalendarEventSource.md) |  | [optional] 
**emitter** | [**Dict[str, IoArgoprojEventsV1alpha1EmitterEventSource]**](IoArgoprojEventsV1alpha1EmitterEventSource.md) |  | [optional] 
**event_bus_name** | **str** |  | [optional] 
**file** | [**Dict[str, IoArgoprojEventsV1alpha1FileEventSource]**](IoArgoprojEventsV1alpha1FileEventSource.md) |  | [optional] 
**generic** | [**Dict[str, IoArgoprojEventsV1alpha1GenericEventSource]**](IoArgoprojEventsV1alpha1GenericEventSource.md) |  | [optional] 
**gerrit** | [**Dict[str, IoArgoprojEventsV1alpha1GerritEventSource]**](IoArgoprojEventsV1alpha1GerritEventSource.md) |  | [optional] 
**github** | [**Dict[str, IoArgoprojEventsV1alpha1GithubEventSource]**](IoArgoprojEventsV1alpha1GithubEventSource.md) |  | [optional] 
**gitlab** | [**Dict[str, IoArgoprojEventsV1alpha1GitlabEventSource]**](IoArgoprojEventsV1alpha1GitlabEventSource.md) |  | [optional] 
**hdfs** | [**Dict[str, IoArgoprojEventsV1alpha1HDFSEventSource]**](IoArgoprojEventsV1alpha1HDFSEventSource.md) |  | [optional] 
**kafka** | [**Dict[str, IoArgoprojEventsV1alpha1KafkaEventSource]**](IoArgoprojEventsV1alpha1KafkaEventSource.md) |  | [optional] 
**minio** | [**Dict[str, IoArgoprojEventsV1alpha1S3Artifact]**](IoArgoprojEventsV1alpha1S3Artifact.md) |  | [optional] 
**mqtt** | [**Dict[str, IoArgoprojEventsV1alpha1MQTTEventSource]**](IoArgoprojEventsV1alpha1MQTTEventSource.md) |  | [optional] 
**nats** | [**Dict[str, IoArgoprojEventsV1alpha1NATSEventsSource]**](IoArgoprojEventsV1alpha1NATSEventsSource.md) |  | [optional] 
**nsq** | [**Dict[str, IoArgoprojEventsV1alpha1NSQEventSource]**](IoArgoprojEventsV1alpha1NSQEventSource.md) |  | [optional] 
**pub_sub** | [**Dict[str, IoArgoprojEventsV1alpha1PubSubEventSource]**](IoArgoprojEventsV1alpha1PubSubEventSource.md) |  | [optional] 
**pulsar** | [**Dict[str, IoArgoprojEventsV1alpha1PulsarEventSource]**](IoArgoprojEventsV1alpha1PulsarEventSource.md) |  | [optional] 
**redis** | [**Dict[str, IoArgoprojEventsV1alpha1RedisEventSource]**](IoArgoprojEventsV1alpha1RedisEventSource.md) |  | [optional] 
**redis_stream** | [**Dict[str, IoArgoprojEventsV1alpha1RedisStreamEventSource]**](IoArgoprojEventsV1alpha1RedisStreamEventSource.md) |  | [optional] 
**replicas** | **int** |  | [optional] 
**resource** | [**Dict[str, IoArgoprojEventsV1alpha1ResourceEventSource]**](IoArgoprojEventsV1alpha1ResourceEventSource.md) |  | [optional] 
**service** | [**IoArgoprojEventsV1alpha1Service**](IoArgoprojEventsV1alpha1Service.md) |  | [optional] 
**sftp** | [**Dict[str, IoArgoprojEventsV1alpha1SFTPEventSource]**](IoArgoprojEventsV1alpha1SFTPEventSource.md) |  | [optional] 
**slack** | [**Dict[str, IoArgoprojEventsV1alpha1SlackEventSource]**](IoArgoprojEventsV1alpha1SlackEventSource.md) |  | [optional] 
**sns** | [**Dict[str, IoArgoprojEventsV1alpha1SNSEventSource]**](IoArgoprojEventsV1alpha1SNSEventSource.md) |  | [optional] 
**sqs** | [**Dict[str, IoArgoprojEventsV1alpha1SQSEventSource]**](IoArgoprojEventsV1alpha1SQSEventSource.md) |  | [optional] 
**storage_grid** | [**Dict[str, IoArgoprojEventsV1alpha1StorageGridEventSource]**](IoArgoprojEventsV1alpha1StorageGridEventSource.md) |  | [optional] 
**stripe** | [**Dict[str, IoArgoprojEventsV1alpha1StripeEventSource]**](IoArgoprojEventsV1alpha1StripeEventSource.md) |  | [optional] 
**template** | [**IoArgoprojEventsV1alpha1Template**](IoArgoprojEventsV1alpha1Template.md) |  | [optional] 
**webhook** | [**Dict[str, IoArgoprojEventsV1alpha1WebhookEventSource]**](IoArgoprojEventsV1alpha1WebhookEventSource.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_event_source_spec import IoArgoprojEventsV1alpha1EventSourceSpec

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1EventSourceSpec from a JSON string
io_argoproj_events_v1alpha1_event_source_spec_instance = IoArgoprojEventsV1alpha1EventSourceSpec.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1EventSourceSpec.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_event_source_spec_dict = io_argoproj_events_v1alpha1_event_source_spec_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1EventSourceSpec from a dict
io_argoproj_events_v1alpha1_event_source_spec_form_dict = io_argoproj_events_v1alpha1_event_source_spec.from_dict(io_argoproj_events_v1alpha1_event_source_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


