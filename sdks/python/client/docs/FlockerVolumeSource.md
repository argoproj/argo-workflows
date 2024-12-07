# FlockerVolumeSource

Represents a Flocker volume mounted by the Flocker agent. One and only one of datasetName and datasetUUID should be set. Flocker volumes do not support ownership management or SELinux relabeling.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**dataset_name** | **str** | datasetName is Name of the dataset stored as metadata -&gt; name on the dataset for Flocker should be considered as deprecated | [optional] 
**dataset_uuid** | **str** | datasetUUID is the UUID of the dataset. This is unique identifier of a Flocker dataset | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


