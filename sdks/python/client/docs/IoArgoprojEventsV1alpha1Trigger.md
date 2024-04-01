# IoArgoprojEventsV1alpha1Trigger


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**at_least_once** | **bool** |  | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**policy** | [**IoArgoprojEventsV1alpha1TriggerPolicy**](IoArgoprojEventsV1alpha1TriggerPolicy.md) |  | [optional] 
**rate_limit** | [**IoArgoprojEventsV1alpha1RateLimit**](IoArgoprojEventsV1alpha1RateLimit.md) |  | [optional] 
**retry_strategy** | [**IoArgoprojEventsV1alpha1Backoff**](IoArgoprojEventsV1alpha1Backoff.md) |  | [optional] 
**template** | [**IoArgoprojEventsV1alpha1TriggerTemplate**](IoArgoprojEventsV1alpha1TriggerTemplate.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_trigger import IoArgoprojEventsV1alpha1Trigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1Trigger from a JSON string
io_argoproj_events_v1alpha1_trigger_instance = IoArgoprojEventsV1alpha1Trigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1Trigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_trigger_dict = io_argoproj_events_v1alpha1_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1Trigger from a dict
io_argoproj_events_v1alpha1_trigger_form_dict = io_argoproj_events_v1alpha1_trigger.from_dict(io_argoproj_events_v1alpha1_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


