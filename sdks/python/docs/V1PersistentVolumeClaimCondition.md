# V1PersistentVolumeClaimCondition

PersistentVolumeClaimCondition contails details about state of pvc
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**last_probe_time** | **datetime** | Last time we probed the condition. | [optional] 
**last_transition_time** | **datetime** | Last time the condition transitioned from one status to another. | [optional] 
**message** | **str** | Human-readable message indicating details about last transition. | [optional] 
**reason** | **str** | Unique, this should be a short, machine understandable string that gives the reason for condition&#39;s last transition. If it reports \&quot;ResizeStarted\&quot; that means the underlying persistent volume is being resized. | [optional] 
**status** | **str** |  | 
**type** | **str** |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


