# PersistentVolumeClaimCondition

PersistentVolumeClaimCondition contails details about state of pvc

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | **str** |  | 
**type** | **str** |    Possible enum values:  - &#x60;\&quot;FileSystemResizePending\&quot;&#x60; - controller resize is finished and a file system resize is pending on node  - &#x60;\&quot;Resizing\&quot;&#x60; - a user trigger resize of pvc has been started | 
**last_probe_time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**last_transition_time** | **datetime** | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional] 
**message** | **str** | Human-readable message indicating details about last transition. | [optional] 
**reason** | **str** | Unique, this should be a short, machine understandable string that gives the reason for condition&#39;s last transition. If it reports \&quot;ResizeStarted\&quot; that means the underlying persistent volume is being resized. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


