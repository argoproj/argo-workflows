# IoArgoprojWorkflowV1alpha1AzureArtifactRepository

AzureArtifactRepository defines the controller configuration for an Azure Blob Storage artifact repository

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**account_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**blob_name_format** | **str** | BlobNameFormat is defines the format of how to store blob names. Can reference workflow variables | [optional] 
**container** | **str** | Container is the container where resources will be stored | 
**endpoint** | **str** | Endpoint is the service url associated with an account. It is most likely \&quot;https://&lt;ACCOUNT_NAME&gt;.blob.core.windows.net\&quot; | 
**use_sdk_creds** | **bool** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_azure_artifact_repository import IoArgoprojWorkflowV1alpha1AzureArtifactRepository

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1AzureArtifactRepository from a JSON string
io_argoproj_workflow_v1alpha1_azure_artifact_repository_instance = IoArgoprojWorkflowV1alpha1AzureArtifactRepository.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1AzureArtifactRepository.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_azure_artifact_repository_dict = io_argoproj_workflow_v1alpha1_azure_artifact_repository_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1AzureArtifactRepository from a dict
io_argoproj_workflow_v1alpha1_azure_artifact_repository_form_dict = io_argoproj_workflow_v1alpha1_azure_artifact_repository.from_dict(io_argoproj_workflow_v1alpha1_azure_artifact_repository_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


