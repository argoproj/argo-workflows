# IoArgoprojEventsV1alpha1EmailTrigger

EmailTrigger refers to the specification of the email notification trigger.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**body** | **str** |  | [optional] 
**var_from** | **str** |  | [optional] 
**host** | **str** | Host refers to the smtp host url to which email is send. | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**port** | **int** |  | [optional] 
**smtp_password** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**subject** | **str** |  | [optional] 
**to** | **List[str]** |  | [optional] 
**username** | **str** |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_email_trigger import IoArgoprojEventsV1alpha1EmailTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1EmailTrigger from a JSON string
io_argoproj_events_v1alpha1_email_trigger_instance = IoArgoprojEventsV1alpha1EmailTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1EmailTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_email_trigger_dict = io_argoproj_events_v1alpha1_email_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1EmailTrigger from a dict
io_argoproj_events_v1alpha1_email_trigger_form_dict = io_argoproj_events_v1alpha1_email_trigger.from_dict(io_argoproj_events_v1alpha1_email_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


