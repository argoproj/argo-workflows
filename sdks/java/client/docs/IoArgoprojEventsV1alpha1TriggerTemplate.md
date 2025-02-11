

# IoArgoprojEventsV1alpha1TriggerTemplate

TriggerTemplate is the template that describes trigger specification.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**argoWorkflow** | [**IoArgoprojEventsV1alpha1ArgoWorkflowTrigger**](IoArgoprojEventsV1alpha1ArgoWorkflowTrigger.md) |  |  [optional]
**awsLambda** | [**IoArgoprojEventsV1alpha1AWSLambdaTrigger**](IoArgoprojEventsV1alpha1AWSLambdaTrigger.md) |  |  [optional]
**azureEventHubs** | [**IoArgoprojEventsV1alpha1AzureEventHubsTrigger**](IoArgoprojEventsV1alpha1AzureEventHubsTrigger.md) |  |  [optional]
**conditions** | **String** |  |  [optional]
**conditionsReset** | [**List&lt;IoArgoprojEventsV1alpha1ConditionsResetCriteria&gt;**](IoArgoprojEventsV1alpha1ConditionsResetCriteria.md) |  |  [optional]
**custom** | [**IoArgoprojEventsV1alpha1CustomTrigger**](IoArgoprojEventsV1alpha1CustomTrigger.md) |  |  [optional]
**http** | [**IoArgoprojEventsV1alpha1HTTPTrigger**](IoArgoprojEventsV1alpha1HTTPTrigger.md) |  |  [optional]
**k8s** | [**IoArgoprojEventsV1alpha1StandardK8STrigger**](IoArgoprojEventsV1alpha1StandardK8STrigger.md) |  |  [optional]
**kafka** | [**IoArgoprojEventsV1alpha1KafkaTrigger**](IoArgoprojEventsV1alpha1KafkaTrigger.md) |  |  [optional]
**log** | [**IoArgoprojEventsV1alpha1LogTrigger**](IoArgoprojEventsV1alpha1LogTrigger.md) |  |  [optional]
**name** | **String** | Name is a unique name of the action to take. |  [optional]
**nats** | [**IoArgoprojEventsV1alpha1NATSTrigger**](IoArgoprojEventsV1alpha1NATSTrigger.md) |  |  [optional]
**openWhisk** | [**IoArgoprojEventsV1alpha1OpenWhiskTrigger**](IoArgoprojEventsV1alpha1OpenWhiskTrigger.md) |  |  [optional]
**pulsar** | [**IoArgoprojEventsV1alpha1PulsarTrigger**](IoArgoprojEventsV1alpha1PulsarTrigger.md) |  |  [optional]
**slack** | [**IoArgoprojEventsV1alpha1SlackTrigger**](IoArgoprojEventsV1alpha1SlackTrigger.md) |  |  [optional]



