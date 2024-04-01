# IoArgoprojEventsV1alpha1EventContext


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**datacontenttype** | **str** | DataContentType - A MIME (RFC2046) string describing the media type of &#x60;data&#x60;. | [optional] 
**id** | **str** | ID of the event; must be non-empty and unique within the scope of the producer. | [optional] 
**source** | **str** | Source - A URI describing the event producer. | [optional] 
**specversion** | **str** | SpecVersion - The version of the CloudEvents specification used by the io.argoproj.workflow.v1alpha1. | [optional] 
**subject** | **str** |  | [optional] 
**time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**type** | **str** | Type - The type of the occurrence which has happened. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_event_context import IoArgoprojEventsV1alpha1EventContext

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1EventContext from a JSON string
io_argoproj_events_v1alpha1_event_context_instance = IoArgoprojEventsV1alpha1EventContext.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1EventContext.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_event_context_dict = io_argoproj_events_v1alpha1_event_context_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1EventContext from a dict
io_argoproj_events_v1alpha1_event_context_form_dict = io_argoproj_events_v1alpha1_event_context.from_dict(io_argoproj_events_v1alpha1_event_context_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


