# IoArgoprojWorkflowV1alpha1S3EncryptionOptions

S3EncryptionOptions used to determine encryption options during s3 operations

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**enable_encryption** | **bool** | EnableEncryption tells the driver to encrypt objects if set to true. If kmsKeyId and serverSideCustomerKeySecret are not set, SSE-S3 will be used | [optional] 
**kms_encryption_context** | **str** | KmsEncryptionContext is a json blob that contains an encryption context. See https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#encrypt_context for more information | [optional] 
**kms_key_id** | **str** | KMSKeyId tells the driver to encrypt the object using the specified KMS Key. | [optional] 
**server_side_customer_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_s3_encryption_options import IoArgoprojWorkflowV1alpha1S3EncryptionOptions

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1S3EncryptionOptions from a JSON string
io_argoproj_workflow_v1alpha1_s3_encryption_options_instance = IoArgoprojWorkflowV1alpha1S3EncryptionOptions.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1S3EncryptionOptions.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_s3_encryption_options_dict = io_argoproj_workflow_v1alpha1_s3_encryption_options_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1S3EncryptionOptions from a dict
io_argoproj_workflow_v1alpha1_s3_encryption_options_form_dict = io_argoproj_workflow_v1alpha1_s3_encryption_options.from_dict(io_argoproj_workflow_v1alpha1_s3_encryption_options_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


