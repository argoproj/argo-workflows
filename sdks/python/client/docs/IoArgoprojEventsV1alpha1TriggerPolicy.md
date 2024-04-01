# IoArgoprojEventsV1alpha1TriggerPolicy


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**k8s** | [**IoArgoprojEventsV1alpha1K8SResourcePolicy**](IoArgoprojEventsV1alpha1K8SResourcePolicy.md) |  | [optional] 
**status** | [**IoArgoprojEventsV1alpha1StatusPolicy**](IoArgoprojEventsV1alpha1StatusPolicy.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_trigger_policy import IoArgoprojEventsV1alpha1TriggerPolicy

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1TriggerPolicy from a JSON string
io_argoproj_events_v1alpha1_trigger_policy_instance = IoArgoprojEventsV1alpha1TriggerPolicy.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1TriggerPolicy.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_trigger_policy_dict = io_argoproj_events_v1alpha1_trigger_policy_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1TriggerPolicy from a dict
io_argoproj_events_v1alpha1_trigger_policy_form_dict = io_argoproj_events_v1alpha1_trigger_policy.from_dict(io_argoproj_events_v1alpha1_trigger_policy_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


