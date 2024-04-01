# IoArgoprojEventsV1alpha1GerritEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth** | [**IoArgoprojEventsV1alpha1BasicAuth**](IoArgoprojEventsV1alpha1BasicAuth.md) |  | [optional] 
**delete_hook_on_finish** | **bool** |  | [optional] 
**events** | **List[str]** |  | [optional] 
**filter** | [**IoArgoprojEventsV1alpha1EventSourceFilter**](IoArgoprojEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**gerrit_base_url** | **str** |  | [optional] 
**hook_name** | **str** |  | [optional] 
**metadata** | **Dict[str, str]** |  | [optional] 
**projects** | **List[str]** | List of project namespace paths like \&quot;whynowy/test\&quot;. | [optional] 
**ssl_verify** | **bool** |  | [optional] 
**webhook** | [**IoArgoprojEventsV1alpha1WebhookContext**](IoArgoprojEventsV1alpha1WebhookContext.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_gerrit_event_source import IoArgoprojEventsV1alpha1GerritEventSource

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1GerritEventSource from a JSON string
io_argoproj_events_v1alpha1_gerrit_event_source_instance = IoArgoprojEventsV1alpha1GerritEventSource.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1GerritEventSource.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_gerrit_event_source_dict = io_argoproj_events_v1alpha1_gerrit_event_source_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1GerritEventSource from a dict
io_argoproj_events_v1alpha1_gerrit_event_source_form_dict = io_argoproj_events_v1alpha1_gerrit_event_source.from_dict(io_argoproj_events_v1alpha1_gerrit_event_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


