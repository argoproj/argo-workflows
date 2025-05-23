

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerTemplate

TriggerTemplate is the template that describes trigger specification.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**argoWorkflow** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ArgoWorkflowTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ArgoWorkflowTrigger.md) |  |  [optional]
**awsLambda** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AWSLambdaTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AWSLambdaTrigger.md) |  |  [optional]
**azureEventHubs** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureEventHubsTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureEventHubsTrigger.md) |  |  [optional]
**azureServiceBus** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureServiceBusTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1AzureServiceBusTrigger.md) |  |  [optional]
**conditions** | **String** |  |  [optional]
**conditionsReset** | [**List&lt;GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ConditionsResetCriteria&gt;**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ConditionsResetCriteria.md) |  |  [optional]
**custom** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1CustomTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1CustomTrigger.md) |  |  [optional]
**email** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EmailTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EmailTrigger.md) |  |  [optional]
**http** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1HTTPTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1HTTPTrigger.md) |  |  [optional]
**k8s** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StandardK8STrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1StandardK8STrigger.md) |  |  [optional]
**kafka** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1KafkaTrigger.md) |  |  [optional]
**log** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1LogTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1LogTrigger.md) |  |  [optional]
**name** | **String** | Name is a unique name of the action to take. |  [optional]
**nats** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1NATSTrigger.md) |  |  [optional]
**openWhisk** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1OpenWhiskTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1OpenWhiskTrigger.md) |  |  [optional]
**pulsar** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PulsarTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PulsarTrigger.md) |  |  [optional]
**slack** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackTrigger**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1SlackTrigger.md) |  |  [optional]



