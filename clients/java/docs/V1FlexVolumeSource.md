

# V1FlexVolumeSource

FlexVolume represents a generic volume resource that is provisioned/attached using an exec based plugin.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**driver** | **String** | Driver is the name of the driver to use for this volume. |  [optional]
**fsType** | **String** |  |  [optional]
**options** | **Map&lt;String, String&gt;** |  |  [optional]
**readOnly** | **Boolean** |  |  [optional]
**secretRef** | [**V1LocalObjectReference**](V1LocalObjectReference.md) |  |  [optional]



