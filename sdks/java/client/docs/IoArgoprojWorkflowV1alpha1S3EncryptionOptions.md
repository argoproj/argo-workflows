

# IoArgoprojWorkflowV1alpha1S3EncryptionOptions

S3EncryptionOptions used to determine encryption options during s3 operations

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enableEncryption** | **Boolean** | EnableEncryption tells the driver to encrypt objects if set to true. If kmsKeyId and serverSideCustomerKeySecret are not set, SSE-S3 will be used |  [optional]
**kmsEncryptionContext** | **String** | KmsEncryptionContext is a json blob that contains an encryption context. See https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#encrypt_context for more information |  [optional]
**kmsKeyId** | **String** | KMSKeyId tells the driver to encrypt the object using the specified KMS Key. |  [optional]
**serverSideCustomerKeySecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]



