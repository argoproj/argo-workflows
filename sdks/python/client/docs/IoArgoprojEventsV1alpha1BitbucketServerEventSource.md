# IoArgoprojEventsV1alpha1BitbucketServerEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_token** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**bitbucketserver_base_url** | **str** |  | [optional] 
**delete_hook_on_finish** | **bool** |  | [optional] 
**events** | **List[str]** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**project_key** | **str** |  | [optional] 
**repositories** | [**List[IoArgoprojEventsV1alpha1BitbucketServerRepository]**](IoArgoprojEventsV1alpha1BitbucketServerRepository.md) |  | [optional] 
**repository_slug** | **str** |  | [optional] 
**tls** | [**IoArgoprojEventsV1alpha1TLSConfig**](IoArgoprojEventsV1alpha1TLSConfig.md) |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 
**webhook_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_bitbucket_server_event_source import IoArgoprojEventsV1alpha1BitbucketServerEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1BitbucketServerEventSource from a JSON string
io_argoproj_events_v1alpha1_bitbucket_server_event_source_instance = IoArgoprojEventsV1alpha1BitbucketServerEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1BitbucketServerEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_bitbucket_server_event_source_dict = io_argoproj_events_v1alpha1_bitbucket_server_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1BitbucketServerEventSource from a dict
io_argoproj_events_v1alpha1_bitbucket_server_event_source_form_dict = io_argoproj_events_v1alpha1_bitbucket_server_event_source.from_dict(io_argoproj_events_v1alpha1_bitbucket_server_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


