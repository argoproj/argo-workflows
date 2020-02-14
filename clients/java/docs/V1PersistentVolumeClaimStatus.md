

# V1PersistentVolumeClaimStatus

PersistentVolumeClaimStatus is the current status of a persistent volume claim.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**accessModes** | **List&lt;String&gt;** |  |  [optional]
**capacity** | [**Map&lt;String, ResourceQuantity&gt;**](ResourceQuantity.md) |  |  [optional]
**conditions** | [**List&lt;V1PersistentVolumeClaimCondition&gt;**](V1PersistentVolumeClaimCondition.md) |  |  [optional]
**phase** | **String** |  |  [optional]



