# IoArgoprojEventsV1alpha1BitbucketEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**IoArgoprojEventsV1alpha1BitbucketAuth**](IoArgoprojEventsV1alpha1BitbucketAuth.md) |  | [optional] 
**delete_hook_on_finish** | **bool** |  | [optional] 
**events** | **List[str]** | Events this webhook is subscribed to. | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**owner** | **str** |  | [optional] 
**project_key** | **str** |  | [optional] 
**repositories** | [**List[IoArgoprojEventsV1alpha1BitbucketRepository]**](IoArgoprojEventsV1alpha1BitbucketRepository.md) |  | [optional] 
**repository_slug** | **str** |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_bitbucket_event_source import IoArgoprojEventsV1alpha1BitbucketEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1BitbucketEventSource from a JSON string
io_argoproj_events_v1alpha1_bitbucket_event_source_instance = IoArgoprojEventsV1alpha1BitbucketEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1BitbucketEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_bitbucket_event_source_dict = io_argoproj_events_v1alpha1_bitbucket_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1BitbucketEventSource from a dict
io_argoproj_events_v1alpha1_bitbucket_event_source_form_dict = io_argoproj_events_v1alpha1_bitbucket_event_source.from_dict(io_argoproj_events_v1alpha1_bitbucket_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


