# IoArgoprojWorkflowV1alpha1ArtifactGCSpec

ArtifactGCSpec specifies the Artifacts that need to be deleted

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**artifacts_by_node** | [**Dict[str, IoArgoprojWorkflowV1alpha1ArtifactNodeSpec]**](IoArgoprojWorkflowV1alpha1ArtifactNodeSpec.md) | ArtifactsByNode maps Node name to information pertaining to Artifacts on that Node | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_artifact_gc_spec import IoArgoprojWorkflowV1alpha1ArtifactGCSpec

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactGCSpec from a JSON string
io_argoproj_workflow_v1alpha1_artifact_gc_spec_instance = IoArgoprojWorkflowV1alpha1ArtifactGCSpec.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1ArtifactGCSpec.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_artifact_gc_spec_dict = io_argoproj_workflow_v1alpha1_artifact_gc_spec_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1ArtifactGCSpec from a dict
io_argoproj_workflow_v1alpha1_artifact_gc_spec_form_dict = io_argoproj_workflow_v1alpha1_artifact_gc_spec.from_dict(io_argoproj_workflow_v1alpha1_artifact_gc_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


