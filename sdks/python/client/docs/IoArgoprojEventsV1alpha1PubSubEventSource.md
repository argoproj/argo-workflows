# IoArgoprojEventsV1alpha1PubSubEventSource

PubSubEventSource refers to event-source for GCP PubSub related events.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**credential_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**delete_subscription_on_finish** | **bool** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**project_id** | **str** |  | [optional] 
**subscription_id** | **str** |  | [optional] 
**topic** | **str** |  | [optional] 
**topic_project_id** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_pub_sub_event_source import IoArgoprojEventsV1alpha1PubSubEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1PubSubEventSource from a JSON string
io_argoproj_events_v1alpha1_pub_sub_event_source_instance = IoArgoprojEventsV1alpha1PubSubEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1PubSubEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_pub_sub_event_source_dict = io_argoproj_events_v1alpha1_pub_sub_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1PubSubEventSource from a dict
io_argoproj_events_v1alpha1_pub_sub_event_source_form_dict = io_argoproj_events_v1alpha1_pub_sub_event_source.from_dict(io_argoproj_events_v1alpha1_pub_sub_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


