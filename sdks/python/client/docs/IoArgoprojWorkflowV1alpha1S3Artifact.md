# IoArgoprojWorkflowV1alpha1S3Artifact

S3Artifact is the location of an S3 artifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**bucket** | **str** | Bucket is the name of the bucket | [optional] 
**ca_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**create_bucket_if_not_present** | [**IoArgoprojWorkflowV1alpha1CreateS3BucketOptions**](IoArgoprojWorkflowV1alpha1CreateS3BucketOptions.md) |  | [optional] 
**encryption_options** | [**IoArgoprojWorkflowV1alpha1S3EncryptionOptions**](IoArgoprojWorkflowV1alpha1S3EncryptionOptions.md) |  | [optional] 
**endpoint** | **str** | Endpoint is the hostname of the bucket endpoint | [optional] 
**insecure** | **bool** | Insecure will connect to the service with TLS | [optional] 
**key** | **str** | Key is the key in the bucket where the artifact resides | [optional] 
**region** | **str** | Region contains the optional bucket region | [optional] 
**role_arn** | **str** | RoleARN is the Amazon Resource Name (ARN) of the role to assume. | [optional] 
**secret_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**use_sdk_creds** | **bool** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_s3_artifact import IoArgoprojWorkflowV1alpha1S3Artifact

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1S3Artifact from a JSON string
io_argoproj_workflow_v1alpha1_s3_artifact_instance = IoArgoprojWorkflowV1alpha1S3Artifact.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1S3Artifact.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_s3_artifact_dict = io_argoproj_workflow_v1alpha1_s3_artifact_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1S3Artifact from a dict
io_argoproj_workflow_v1alpha1_s3_artifact_form_dict = io_argoproj_workflow_v1alpha1_s3_artifact.from_dict(io_argoproj_workflow_v1alpha1_s3_artifact_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


