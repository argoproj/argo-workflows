# IoArgoprojWorkflowV1alpha1AzureArtifact

AzureArtifact is the location of a an Azure Storage artifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**account_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**blob** | **str** | Blob is the blob name (i.e., path) in the container where the artifact resides | 
**container** | **str** | Container is the container where resources will be stored | 
**endpoint** | **str** | Endpoint is the service url associated with an account. It is most likely \&quot;https://&lt;ACCOUNT_NAME&gt;.blob.core.windows.net\&quot; | 
**use_sdk_creds** | **bool** | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_azure_artifact import IoArgoprojWorkflowV1alpha1AzureArtifact

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1AzureArtifact from a JSON string
io_argoproj_workflow_v1alpha1_azure_artifact_instance = IoArgoprojWorkflowV1alpha1AzureArtifact.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1AzureArtifact.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_azure_artifact_dict = io_argoproj_workflow_v1alpha1_azure_artifact_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1AzureArtifact from a dict
io_argoproj_workflow_v1alpha1_azure_artifact_form_dict = io_argoproj_workflow_v1alpha1_azure_artifact.from_dict(io_argoproj_workflow_v1alpha1_azure_artifact_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


