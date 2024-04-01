# IoArgoprojWorkflowV1alpha1ArtifactNodeSpec

ArtifactNodeSpec specifies the Artifacts that need to be deleted for a given Node

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**archive_location** | [**IoArgoprojWorkflowV1alpha1ArtifactLocation**](IoArgoprojWorkflowV1alpha1ArtifactLocation.md) |  | [optional] 
**artifacts** | [**Dict[str, IoArgoprojWorkflowV1alpha1Artifact]**](IoArgoprojWorkflowV1alpha1Artifact.md) | Artifacts maps artifact name to Artifact description | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_artifact_node_spec import IoArgoprojWorkflowV1alpha1ArtifactNodeSpec

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactNodeSpec from a JSON string
io_argoproj_workflow_v1alpha1_artifact_node_spec_instance = IoArgoprojWorkflowV1alpha1ArtifactNodeSpec.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArtifactNodeSpec.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_artifact_node_spec_dict = io_argoproj_workflow_v1alpha1_artifact_node_spec_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactNodeSpec from a dict
io_argoproj_workflow_v1alpha1_artifact_node_spec_form_dict = io_argoproj_workflow_v1alpha1_artifact_node_spec.from_dict(io_argoproj_workflow_v1alpha1_artifact_node_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


