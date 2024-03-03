# IoArgoprojWorkflowV1alpha1RetryConfig

RetryConfig defines how to retry a workflow

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**node_field_selector** | **str** | NodeFieldSelector selects nodes to reset | [optional] 
**parameters** | **[str]** | Parameters are a list of parameters passed | [optional] 
**restart_successful** | **bool** | RestartSuccessful defines whether or not to retry succeeded node | [optional] 
**retried** | **bool** | Retried tracks whether or not this workflow was retried by RetryConfig | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


