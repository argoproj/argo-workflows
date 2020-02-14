

# V1QuobyteVolumeSource

Represents a Quobyte mount that lasts the lifetime of a pod. Quobyte volumes do not support ownership management or SELinux relabeling.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**group** | **String** |  |  [optional]
**readOnly** | **Boolean** |  |  [optional]
**registry** | **String** |  |  [optional]
**tenant** | **String** |  |  [optional]
**user** | **String** |  |  [optional]
**volume** | **String** | Volume is a string that references an already created Quobyte volume by name. |  [optional]



