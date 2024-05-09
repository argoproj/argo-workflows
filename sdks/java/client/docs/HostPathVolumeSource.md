

# HostPathVolumeSource

Represents a host path mapped into a pod. Host path volumes do not support ownership management or SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**path** | **String** | path of the directory on the host. If the path is a symlink, it will follow the link to the real path. More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath | 
**type** | **String** | type for HostPath Volume Defaults to \&quot;\&quot; More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath |  [optional]



