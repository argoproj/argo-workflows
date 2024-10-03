

# IoArgoprojWorkflowV1alpha1StopStrategy

StopStrategy defines if the CronWorkflow should stop scheduling based on an expression. v3.6 and after

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**expression** | **String** | Expression is an expression that stops scheduling workflows when true. Use the variables &#x60;io.argoproj.workflow.v1alpha1.failed&#x60; or &#x60;cronio.argoproj.REPLACEME.v1alpha1.succeeded&#x60; to access the number of failed or successful child workflows. v3.6 and after | 



