# IoArgoprojEventsV1alpha1TriggerTemplate

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**argo_workflow** | Option<[**crate::models::IoArgoprojEventsV1alpha1ArgoWorkflowTrigger**](io.argoproj.events.v1alpha1.ArgoWorkflowTrigger.md)> |  | [optional]
**aws_lambda** | Option<[**crate::models::IoArgoprojEventsV1alpha1AwsLambdaTrigger**](io.argoproj.events.v1alpha1.AWSLambdaTrigger.md)> |  | [optional]
**azure_event_hubs** | Option<[**crate::models::IoArgoprojEventsV1alpha1AzureEventHubsTrigger**](io.argoproj.events.v1alpha1.AzureEventHubsTrigger.md)> |  | [optional]
**conditions** | Option<**String**> |  | [optional]
**conditions_reset** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1ConditionsResetCriteria>**](io.argoproj.events.v1alpha1.ConditionsResetCriteria.md)> |  | [optional]
**custom** | Option<[**crate::models::IoArgoprojEventsV1alpha1CustomTrigger**](io.argoproj.events.v1alpha1.CustomTrigger.md)> |  | [optional]
**http** | Option<[**crate::models::IoArgoprojEventsV1alpha1HttpTrigger**](io.argoproj.events.v1alpha1.HTTPTrigger.md)> |  | [optional]
**k8s** | Option<[**crate::models::IoArgoprojEventsV1alpha1StandardK8STrigger**](io.argoproj.events.v1alpha1.StandardK8STrigger.md)> |  | [optional]
**kafka** | Option<[**crate::models::IoArgoprojEventsV1alpha1KafkaTrigger**](io.argoproj.events.v1alpha1.KafkaTrigger.md)> |  | [optional]
**log** | Option<[**crate::models::IoArgoprojEventsV1alpha1LogTrigger**](io.argoproj.events.v1alpha1.LogTrigger.md)> |  | [optional]
**name** | Option<**String**> | Name is a unique name of the action to take. | [optional]
**nats** | Option<[**crate::models::IoArgoprojEventsV1alpha1NatsTrigger**](io.argoproj.events.v1alpha1.NATSTrigger.md)> |  | [optional]
**open_whisk** | Option<[**crate::models::IoArgoprojEventsV1alpha1OpenWhiskTrigger**](io.argoproj.events.v1alpha1.OpenWhiskTrigger.md)> |  | [optional]
**pulsar** | Option<[**crate::models::IoArgoprojEventsV1alpha1PulsarTrigger**](io.argoproj.events.v1alpha1.PulsarTrigger.md)> |  | [optional]
**slack** | Option<[**crate::models::IoArgoprojEventsV1alpha1SlackTrigger**](io.argoproj.events.v1alpha1.SlackTrigger.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


