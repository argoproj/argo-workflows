# IoArgoprojEventsV1alpha1TriggerTemplate

TriggerTemplate is the template that describes trigger specification.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**argo_workflow** | [**IoArgoprojEventsV1alpha1ArgoWorkflowTrigger**](IoArgoprojEventsV1alpha1ArgoWorkflowTrigger.md) |  | [optional] 
**aws_lambda** | [**IoArgoprojEventsV1alpha1AWSLambdaTrigger**](IoArgoprojEventsV1alpha1AWSLambdaTrigger.md) |  | [optional] 
**azure_event_hubs** | [**IoArgoprojEventsV1alpha1AzureEventHubsTrigger**](IoArgoprojEventsV1alpha1AzureEventHubsTrigger.md) |  | [optional] 
**conditions** | **str** |  | [optional] 
**conditions_reset** | [**[IoArgoprojEventsV1alpha1ConditionsResetCriteria]**](IoArgoprojEventsV1alpha1ConditionsResetCriteria.md) |  | [optional] 
**custom** | [**IoArgoprojEventsV1alpha1CustomTrigger**](IoArgoprojEventsV1alpha1CustomTrigger.md) |  | [optional] 
**http** | [**IoArgoprojEventsV1alpha1HTTPTrigger**](IoArgoprojEventsV1alpha1HTTPTrigger.md) |  | [optional] 
**k8s** | [**IoArgoprojEventsV1alpha1StandardK8STrigger**](IoArgoprojEventsV1alpha1StandardK8STrigger.md) |  | [optional] 
**kafka** | [**IoArgoprojEventsV1alpha1KafkaTrigger**](IoArgoprojEventsV1alpha1KafkaTrigger.md) |  | [optional] 
**log** | [**IoArgoprojEventsV1alpha1LogTrigger**](IoArgoprojEventsV1alpha1LogTrigger.md) |  | [optional] 
**name** | **str** | Name is a unique name of the action to take. | [optional] 
**nats** | [**IoArgoprojEventsV1alpha1NATSTrigger**](IoArgoprojEventsV1alpha1NATSTrigger.md) |  | [optional] 
**open_whisk** | [**IoArgoprojEventsV1alpha1OpenWhiskTrigger**](IoArgoprojEventsV1alpha1OpenWhiskTrigger.md) |  | [optional] 
**pulsar** | [**IoArgoprojEventsV1alpha1PulsarTrigger**](IoArgoprojEventsV1alpha1PulsarTrigger.md) |  | [optional] 
**slack** | [**IoArgoprojEventsV1alpha1SlackTrigger**](IoArgoprojEventsV1alpha1SlackTrigger.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


