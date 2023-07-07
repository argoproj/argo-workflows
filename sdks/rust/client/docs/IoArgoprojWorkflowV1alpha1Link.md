# IoArgoprojWorkflowV1alpha1Link

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**name** | **String** | The name of the link, E.g. \"Workflow Logs\" or \"Pod Logs\" | 
**scope** | **String** | \"workflow\", \"pod\", \"pod-logs\", \"event-source-logs\", \"sensor-logs\", \"workflow-list\" or \"chat\" | 
**url** | **String** | The URL. Can contain \"${metadata.namespace}\", \"${metadata.name}\", \"${status.startedAt}\", \"${status.finishedAt}\" or any other element in workflow yaml, e.g. \"${io.argoproj.workflow.v1alpha1.metadata.annotations.userDefinedKey}\" | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


