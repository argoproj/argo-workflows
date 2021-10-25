# IoArgoprojWorkflowV1alpha1OSSLifecycleRule

OSSLifecycleRule specifies how to manage bucket's lifecycle

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**mark_deletion_after_days** | **int** | MarkDeletionAfterDays is the number of days before we delete objects in the bucket | [optional] 
**mark_infrequent_access_after_days** | **int** | MarkInfrequentAccessAfterDays is the number of days before we convert the objects in the bucket to Infrequent Access (IA) storage type | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


