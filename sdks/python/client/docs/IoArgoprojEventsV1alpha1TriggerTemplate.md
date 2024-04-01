# IoArgoprojEventsV1alpha1TriggerTemplate

TriggerTemplate is the template that describes trigger specification.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**argo_workflow** | [**IoArgoprojEventsV1alpha1ArgoWorkflowTrigger**](IoArgoprojEventsV1alpha1ArgoWorkflowTrigger.md) |  | [optional] 
**aws_lambda** | [**IoArgoprojEventsV1alpha1AWSLambdaTrigger**](IoArgoprojEventsV1alpha1AWSLambdaTrigger.md) |  | [optional] 
**azure_event_hubs** | [**IoArgoprojEventsV1alpha1AzureEventHubsTrigger**](IoArgoprojEventsV1alpha1AzureEventHubsTrigger.md) |  | [optional] 
**azure_service_bus** | [**IoArgoprojEventsV1alpha1AzureServiceBusTrigger**](IoArgoprojEventsV1alpha1AzureServiceBusTrigger.md) |  | [optional] 
**conditions** | **str** |  | [optional] 
**conditions_reset** | [**List[IoArgoprojEventsV1alpha1ConditionsResetCriteria]**](IoArgoprojEventsV1alpha1ConditionsResetCriteria.md) |  | [optional] 
**custom** | [**IoArgoprojEventsV1alpha1CustomTrigger**](IoArgoprojEventsV1alpha1CustomTrigger.md) |  | [optional] 
**email** | [**IoArgoprojEventsV1alpha1EmailTrigger**](IoArgoprojEventsV1alpha1EmailTrigger.md) |  | [optional] 
**http** | [**IoArgoprojEventsV1alpha1HTTPTrigger**](IoArgoprojEventsV1alpha1HTTPTrigger.md) |  | [optional] 
**k8s** | [**IoArgoprojEventsV1alpha1StandardK8STrigger**](IoArgoprojEventsV1alpha1StandardK8STrigger.md) |  | [optional] 
**kafka** | [**IoArgoprojEventsV1alpha1KafkaTrigger**](IoArgoprojEventsV1alpha1KafkaTrigger.md) |  | [optional] 
**log** | [**IoArgoprojEventsV1alpha1LogTrigger**](IoArgoprojEventsV1alpha1LogTrigger.md) |  | [optional] 
**name** | **str** | Name is a unique name of the action to take. | [optional] 
**nats** | [**IoArgoprojEventsV1alpha1NATSTrigger**](IoArgoprojEventsV1alpha1NATSTrigger.md) |  | [optional] 
**open_whisk** | [**IoArgoprojEventsV1alpha1OpenWhiskTrigger**](IoArgoprojEventsV1alpha1OpenWhiskTrigger.md) |  | [optional] 
**pulsar** | [**IoArgoprojEventsV1alpha1PulsarTrigger**](IoArgoprojEventsV1alpha1PulsarTrigger.md) |  | [optional] 
**slack** | [**IoArgoprojEventsV1alpha1SlackTrigger**](IoArgoprojEventsV1alpha1SlackTrigger.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_trigger_template import IoArgoprojEventsV1alpha1TriggerTemplate

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1TriggerTemplate from a JSON string
io_argoproj_events_v1alpha1_trigger_template_instance = IoArgoprojEventsV1alpha1TriggerTemplate.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1TriggerTemplate.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_trigger_template_dict = io_argoproj_events_v1alpha1_trigger_template_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1TriggerTemplate from a dict
io_argoproj_events_v1alpha1_trigger_template_form_dict = io_argoproj_events_v1alpha1_trigger_template.from_dict(io_argoproj_events_v1alpha1_trigger_template_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


