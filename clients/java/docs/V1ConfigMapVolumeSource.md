

# V1ConfigMapVolumeSource

Adapts a ConfigMap into a volume.  The contents of the target ConfigMap's Data field will be presented in a volume as files using the keys in the Data field as the file names, unless the items element is populated with specific mappings of keys to paths. ConfigMap volumes support ownership management and SELinux relabeling.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**defaultMode** | **Integer** |  |  [optional]
**items** | [**List&lt;V1KeyToPath&gt;**](V1KeyToPath.md) |  |  [optional]
**localObjectReference** | [**V1LocalObjectReference**](V1LocalObjectReference.md) |  |  [optional]
**optional** | **Boolean** |  |  [optional]



