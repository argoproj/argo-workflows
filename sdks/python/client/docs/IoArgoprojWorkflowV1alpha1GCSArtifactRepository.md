# IoArgoprojWorkflowV1alpha1GCSArtifactRepository

GCSArtifactRepository defines the controller configuration for a GCS artifact repository
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**bucket** | **str** | Bucket is the name of the bucket | [optional] 
**key_format** | **str** | KeyFormat is defines the format of how to store keys. Can reference workflow variables | [optional] 
**service_account_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


