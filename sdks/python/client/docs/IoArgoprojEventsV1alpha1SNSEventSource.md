# IoArgoprojEventsV1alpha1SNSEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_key** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**endpoint** | **str** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**region** | **str** |  | [optional] 
**role_arn** | **str** |  | [optional] 
**secret_key** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**topic_arn** | **str** |  | [optional] 
**validate_signature** | **bool** |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_sns_event_source import IoArgoprojEventsV1alpha1SNSEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1SNSEventSource from a JSON string
io_argoproj_events_v1alpha1_sns_event_source_instance = IoArgoprojEventsV1alpha1SNSEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1SNSEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_sns_event_source_dict = io_argoproj_events_v1alpha1_sns_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1SNSEventSource from a dict
io_argoproj_events_v1alpha1_sns_event_source_form_dict = io_argoproj_events_v1alpha1_sns_event_source.from_dict(io_argoproj_events_v1alpha1_sns_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


