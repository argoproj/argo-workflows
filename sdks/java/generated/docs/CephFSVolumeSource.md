

# CephFSVolumeSource

Represents a Ceph Filesystem mount that lasts the lifetime of a pod Cephfs volumes do not support ownership management or SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**monitors** | **List&lt;String&gt;** | Required: Monitors is a collection of Ceph monitors More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it | 
**path** | **String** | Optional: Used as the mounted root, rather than the full Ceph tree, default is / |  [optional]
**readOnly** | **Boolean** | Optional: Defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it |  [optional]
**secretFile** | **String** | Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it |  [optional]
**secretRef** | [**io.kubernetes.client.openapi.models.V1LocalObjectReference**](io.kubernetes.client.openapi.models.V1LocalObjectReference.md) |  |  [optional]
**user** | **String** | Optional: User is the rados user name, default is admin More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it |  [optional]



