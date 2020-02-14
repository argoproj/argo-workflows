

# V1SecretVolumeSource

Adapts a Secret into a volume.  The contents of the target Secret's Data field will be presented in a volume as files using the keys in the Data field as the file names. Secret volumes support ownership management and SELinux relabeling.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**defaultMode** | **Integer** |  |  [optional]
**items** | [**List&lt;V1KeyToPath&gt;**](V1KeyToPath.md) |  |  [optional]
**optional** | **Boolean** |  |  [optional]
**secretName** | **String** |  |  [optional]



