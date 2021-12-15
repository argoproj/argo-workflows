# IoArgoprojWorkflowV1alpha1S3EncryptionOptions

S3EncryptionOptions used to determine encryption options during s3 operations

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enable_encryption** | **bool** | EnableEncryption tells the driver to encrypt objects if set to true. If kmsKeyId and serverSideCustomerKeySecret are not set, SSE-S3 will be used | [optional] 
**kms_encryption_context** | **str** | KmsEncryptionContext is a json blob that contains an encryption context. See https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#encrypt_context for more information | [optional] 
**kms_key_id** | **str** | KMSKeyId tells the driver to encrypt the object using the specified KMS Key. | [optional] 
**server_side_customer_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


