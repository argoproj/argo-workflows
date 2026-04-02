

# PersistentVolumeClaimCondition

PersistentVolumeClaimCondition contains details about state of pvc

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**lastProbeTime** | **java.time.Instant** |  |  [optional]
**lastTransitionTime** | **java.time.Instant** |  |  [optional]
**message** | **String** | message is the human-readable message indicating details about last transition. |  [optional]
**reason** | **String** | reason is a unique, this should be a short, machine understandable string that gives the reason for condition&#39;s last transition. If it reports \&quot;Resizing\&quot; that means the underlying persistent volume is being resized. |  [optional]
**status** | **String** | Status is the status of the condition. Can be True, False, Unknown. More info: https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#:~:text&#x3D;state%20of%20pvc-,conditions.status,-(string)%2C%20required | 
**type** | **String** | Type is the type of the condition. More info: https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#:~:text&#x3D;set%20to%20%27ResizeStarted%27.-,PersistentVolumeClaimCondition,-contains%20details%20about | 



