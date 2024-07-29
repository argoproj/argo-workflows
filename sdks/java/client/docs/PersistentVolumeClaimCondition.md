

# PersistentVolumeClaimCondition

PersistentVolumeClaimCondition contains details about state of pvc

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**lastProbeTime** | **java.time.Instant** |  |  [optional]
**lastTransitionTime** | **java.time.Instant** |  |  [optional]
**message** | **String** | message is the human-readable message indicating details about last transition. |  [optional]
**reason** | **String** | reason is a unique, this should be a short, machine understandable string that gives the reason for condition&#39;s last transition. If it reports \&quot;Resizing\&quot; that means the underlying persistent volume is being resized. |  [optional]
**status** | **String** |  | 
**type** | **String** |  | 



