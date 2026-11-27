

# IoArgoprojWorkflowV1alpha1StopStrategy

StopStrategy defines if the CronWorkflow should stop scheduling based on an expression.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**expression** | **String** | Expression is an expression that stops scheduling workflows when true. Use the variables &#x60;cronworkflow&#x60;.&#x60;failed&#x60; or &#x60;cronworkflow&#x60;.&#x60;succeeded&#x60; to access the number of failed or successful child workflows. | 



