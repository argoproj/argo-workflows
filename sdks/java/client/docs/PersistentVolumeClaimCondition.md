

# PersistentVolumeClaimCondition

PersistentVolumeClaimCondition contails details about state of pvc

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**lastProbeTime** | **java.time.Instant** |  |  [optional]
**lastTransitionTime** | **java.time.Instant** |  |  [optional]
**message** | **String** | Human-readable message indicating details about last transition. |  [optional]
**reason** | **String** | Unique, this should be a short, machine understandable string that gives the reason for condition&#39;s last transition. If it reports \&quot;ResizeStarted\&quot; that means the underlying persistent volume is being resized. |  [optional]
**status** | **String** |  | 
**type** | [**TypeEnum**](#TypeEnum) |    Possible enum values:  - &#x60;\&quot;FileSystemResizePending\&quot;&#x60; - controller resize is finished and a file system resize is pending on node  - &#x60;\&quot;Resizing\&quot;&#x60; - a user trigger resize of pvc has been started | 



## Enum: TypeEnum

Name | Value
---- | -----
FILESYSTEMRESIZEPENDING | &quot;FileSystemResizePending&quot;
RESIZING | &quot;Resizing&quot;



