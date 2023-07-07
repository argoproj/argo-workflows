# IoArgoprojWorkflowV1alpha1CronWorkflowStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**active** | [**Vec<crate::models::ObjectReference>**](ObjectReference.md) | Active is a list of active workflows stemming from this CronWorkflow | 
**conditions** | [**Vec<crate::models::IoArgoprojWorkflowV1alpha1Condition>**](io.argoproj.workflow.v1alpha1.Condition.md) | Conditions is a list of conditions the CronWorkflow may have | 
**last_scheduled_time** | **String** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


