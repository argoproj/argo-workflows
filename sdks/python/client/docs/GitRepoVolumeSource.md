# GitRepoVolumeSource

Represents a volume that is populated with the contents of a git repository. Git repo volumes do not support ownership management. Git repo volumes support SELinux relabeling.  DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount an EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir into the Pod's container.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**directory** | **str** | Target directory name. Must not contain or start with &#39;..&#39;.  If &#39;.&#39; is supplied, the volume directory will be the git repository.  Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name. | [optional] 
**repository** | **str** | Repository URL | 
**revision** | **str** | Commit hash for the specified revision. | [optional] 

## Example

```python
from argo_workflows.models.git_repo_volume_source import GitRepoVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of GitRepoVolumeSource from a JSON string
git_repo_volume_source_instance = GitRepoVolumeSource.from_json(json)
# print the JSON string representation of the object
print(GitRepoVolumeSource.to_json())

# convert the object into a dict
git_repo_volume_source_dict = git_repo_volume_source_instance.to_dict()
# create an instance of GitRepoVolumeSource from a dict
git_repo_volume_source_form_dict = git_repo_volume_source.from_dict(git_repo_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


