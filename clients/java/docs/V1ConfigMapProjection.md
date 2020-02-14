

# V1ConfigMapProjection

Adapts a ConfigMap into a projected volume.  The contents of the target ConfigMap's Data field will be presented in a projected volume as files using the keys in the Data field as the file names, unless the items element is populated with specific mappings of keys to paths. Note that this is identical to a configmap volume source without the default mode.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List&lt;V1KeyToPath&gt;**](V1KeyToPath.md) |  |  [optional]
**localObjectReference** | [**V1LocalObjectReference**](V1LocalObjectReference.md) |  |  [optional]
**optional** | **Boolean** |  |  [optional]



