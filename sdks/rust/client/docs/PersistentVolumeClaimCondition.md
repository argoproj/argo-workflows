# PersistentVolumeClaimCondition

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**last_probe_time** | Option<**String**> | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional]
**last_transition_time** | Option<**String**> | Time is a wrapper around time.Time which supports correct marshaling to YAML and JSON.  Wrappers are provided for many of the factory methods that the time package offers. | [optional]
**message** | Option<**String**> | Human-readable message indicating details about last transition. | [optional]
**reason** | Option<**String**> | Unique, this should be a short, machine understandable string that gives the reason for condition's last transition. If it reports \"ResizeStarted\" that means the underlying persistent volume is being resized. | [optional]
**status** | **String** |  | 
**_type** | **String** |    Possible enum values:  - `\"FileSystemResizePending\"` - controller resize is finished and a file system resize is pending on node  - `\"Resizing\"` - a user trigger resize of pvc has been started | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


