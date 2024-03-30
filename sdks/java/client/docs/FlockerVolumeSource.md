

# FlockerVolumeSource

Represents a Flocker volume mounted by the Flocker agent. One and only one of datasetName and datasetUUID should be set. Flocker volumes do not support ownership management or SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**datasetName** | **String** | datasetName is Name of the dataset stored as metadata -&gt; name on the dataset for Flocker should be considered as deprecated |  [optional]
**datasetUUID** | **String** | datasetUUID is the UUID of the dataset. This is unique identifier of a Flocker dataset |  [optional]



