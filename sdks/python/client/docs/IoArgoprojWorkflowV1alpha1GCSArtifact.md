# IoArgoprojWorkflowV1alpha1GCSArtifact

GCSArtifact is the location of a GCS artifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**bucket** | **str** | Bucket is the name of the bucket | [optional] 
**key** | **str** | Key is the path in the bucket where the artifact resides | 
**service_account_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_gcs_artifact import IoArgoprojWorkflowV1alpha1GCSArtifact

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1GCSArtifact from a JSON string
io_argoproj_workflow_v1alpha1_gcs_artifact_instance = IoArgoprojWorkflowV1alpha1GCSArtifact.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1GCSArtifact.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_gcs_artifact_dict = io_argoproj_workflow_v1alpha1_gcs_artifact_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1GCSArtifact from a dict
io_argoproj_workflow_v1alpha1_gcs_artifact_form_dict = io_argoproj_workflow_v1alpha1_gcs_artifact.from_dict(io_argoproj_workflow_v1alpha1_gcs_artifact_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


