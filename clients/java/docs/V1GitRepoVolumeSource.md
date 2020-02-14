

# V1GitRepoVolumeSource

Represents a volume that is populated with the contents of a git repository. Git repo volumes do not support ownership management. Git repo volumes support SELinux relabeling.  DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount an EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir into the Pod's container.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**directory** | **String** |  |  [optional]
**repository** | **String** |  |  [optional]
**revision** | **String** |  |  [optional]



