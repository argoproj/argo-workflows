# IoArgoprojWorkflowV1alpha1GitArtifact

GitArtifact is the location of an git artifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**branch** | **str** | Branch is the branch to fetch when &#x60;SingleBranch&#x60; is enabled | [optional] 
**depth** | **int** | Depth specifies clones/fetches should be shallow and include the given number of commits from the branch tip | [optional] 
**disable_submodules** | **bool** | DisableSubmodules disables submodules during git clone | [optional] 
**fetch** | **List[str]** | Fetch specifies a number of refs that should be fetched before checkout | [optional] 
**insecure_ignore_host_key** | **bool** | InsecureIgnoreHostKey disables SSH strict host key checking during git clone | [optional] 
**password_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**repo** | **str** | Repo is the git repository | 
**revision** | **str** | Revision is the git commit, tag, branch to checkout | [optional] 
**single_branch** | **bool** | SingleBranch enables single branch clone, using the &#x60;branch&#x60; parameter | [optional] 
**ssh_private_key_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**username_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_git_artifact import IoArgoprojWorkflowV1alpha1GitArtifact

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1GitArtifact from a JSON string
io_argoproj_workflow_v1alpha1_git_artifact_instance = IoArgoprojWorkflowV1alpha1GitArtifact.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1GitArtifact.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_git_artifact_dict = io_argoproj_workflow_v1alpha1_git_artifact_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1GitArtifact from a dict
io_argoproj_workflow_v1alpha1_git_artifact_form_dict = io_argoproj_workflow_v1alpha1_git_artifact.from_dict(io_argoproj_workflow_v1alpha1_git_artifact_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


