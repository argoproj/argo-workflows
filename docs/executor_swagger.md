


# The API for an executor plugin.
  

## Informations

### Version

0.0.1

## Content negotiation

### URI Schemes
  * http

### Consumes
  * application/json

### Produces
  * application/json

## All endpoints

###  operations

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| POST | /api/v1/template.execute | [execute template](#execute-template) |  |
  


## Paths

### <span id="execute-template"></span> execute template (*executeTemplate*)

```
POST /api/v1/template.execute
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| Body | `body` | [ExecuteTemplateArgs](#execute-template-args) | `models.ExecuteTemplateArgs` | | ✓ | |  |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#execute-template-200) | OK |  |  | [schema](#execute-template-200-schema) |

#### Responses


##### <span id="execute-template-200"></span> 200
Status: OK

###### <span id="execute-template-200-schema"></span> Schema
   
  

[ExecuteTemplateReply](#execute-template-reply)

## Models

### <span id="a-w-s-elastic-block-store-volume-source"></span> AWSElasticBlockStoreVolumeSource


> An AWS EBS disk must exist before mounting to a container. The disk
must also be in the same AWS zone as the kubelet. An AWS EBS disk
can only be mounted as read/write once. AWS EBS volumes support
ownership management and SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fsType | string| `string` |  | | fsType is the filesystem type of the volume that you want to mount.</br>Tip: Ensure that the filesystem type is supported by the host operating system.</br>Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore</br>TODO: how do we prevent errors in the filesystem from compromising the machine</br>+optional |  |
| partition | int32 (formatted integer)| `int32` |  | | partition is the partition in the volume that you want to mount.</br>If omitted, the default is to mount by volume name.</br>Examples: For volume /dev/sda1, you specify the partition as "1".</br>Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty).</br>+optional |  |
| readOnly | boolean| `bool` |  | | readOnly value true will force the readOnly setting in VolumeMounts.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore</br>+optional |  |
| volumeID | string| `string` |  | | volumeID is unique ID of the persistent disk resource in AWS (Amazon EBS volume).</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore |  |



### <span id="affinity"></span> Affinity


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| nodeAffinity | [NodeAffinity](#node-affinity)| `NodeAffinity` |  | |  |  |
| podAffinity | [PodAffinity](#pod-affinity)| `PodAffinity` |  | |  |  |
| podAntiAffinity | [PodAntiAffinity](#pod-anti-affinity)| `PodAntiAffinity` |  | |  |  |



### <span id="amount"></span> Amount


> +kubebuilder:validation:Type=number
  



[interface{}](#interface)

### <span id="any-string"></span> AnyString


> It will unmarshall int64, int32, float64, float32, boolean, a plain string and represents it as string.
It will marshall back to string - marshalling is not symmetric.
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| AnyString | string| string | | It will unmarshall int64, int32, float64, float32, boolean, a plain string and represents it as string.</br>It will marshall back to string - marshalling is not symmetric. |  |



### <span id="app-armor-profile"></span> AppArmorProfile


> +union
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| localhostProfile | string| `string` |  | | localhostProfile indicates a profile loaded on the node that should be used.</br>The profile must be preconfigured on the node to work.</br>Must match the loaded name of the profile.</br>Must be set if and only if type is "Localhost".</br>+optional |  |
| type | [AppArmorProfileType](#app-armor-profile-type)| `AppArmorProfileType` |  | |  |  |



### <span id="app-armor-profile-type"></span> AppArmorProfileType


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| AppArmorProfileType | string| string | | +enum |  |



### <span id="archive-strategy"></span> ArchiveStrategy


> ArchiveStrategy describes how to archive files/directory when saving artifacts
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| none | [NoneStrategy](#none-strategy)| `NoneStrategy` |  | |  |  |
| tar | [TarStrategy](#tar-strategy)| `TarStrategy` |  | |  |  |
| zip | [ZipStrategy](#zip-strategy)| `ZipStrategy` |  | |  |  |



### <span id="arguments"></span> Arguments


> Arguments to a template
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| artifacts | [Artifacts](#artifacts)| `Artifacts` |  | |  |  |
| parameters | [][Parameter](#parameter)| `[]*Parameter` |  | | Parameters is the list of parameters to pass to the template or workflow</br>+patchStrategy=merge</br>+patchMergeKey=name |  |



### <span id="artifact"></span> Artifact


> Artifact indicates an artifact to place at a specified path
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| archive | [ArchiveStrategy](#archive-strategy)| `ArchiveStrategy` |  | |  |  |
| archiveLogs | boolean| `bool` |  | | ArchiveLogs indicates if the container logs should be archived |  |
| artifactGC | [ArtifactGC](#artifact-g-c)| `ArtifactGC` |  | |  |  |
| artifactory | [ArtifactoryArtifact](#artifactory-artifact)| `ArtifactoryArtifact` |  | |  |  |
| azure | [AzureArtifact](#azure-artifact)| `AzureArtifact` |  | |  |  |
| deleted | boolean| `bool` |  | | Has this been deleted? |  |
| from | string| `string` |  | | From allows an artifact to reference an artifact from a previous step |  |
| fromExpression | string| `string` |  | | FromExpression, if defined, is evaluated to specify the value for the artifact |  |
| gcs | [GCSArtifact](#g-c-s-artifact)| `GCSArtifact` |  | |  |  |
| git | [GitArtifact](#git-artifact)| `GitArtifact` |  | |  |  |
| globalName | string| `string` |  | | GlobalName exports an output artifact to the global scope, making it available as</br>'{{workflow.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts |  |
| hdfs | [HDFSArtifact](#h-d-f-s-artifact)| `HDFSArtifact` |  | |  |  |
| http | [HTTPArtifact](#http-artifact)| `HTTPArtifact` |  | |  |  |
| mode | int32 (formatted integer)| `int32` |  | | mode bits to use on this file, must be a value between 0 and 0777</br>set when loading input artifacts. |  |
| name | string| `string` |  | | name of the artifact. must be unique within a template's inputs/outputs. |  |
| optional | boolean| `bool` |  | | Make Artifacts optional, if Artifacts doesn't generate or exist |  |
| oss | [OSSArtifact](#o-s-s-artifact)| `OSSArtifact` |  | |  |  |
| path | string| `string` |  | | Path is the container path to the artifact |  |
| raw | [RawArtifact](#raw-artifact)| `RawArtifact` |  | |  |  |
| recurseMode | boolean| `bool` |  | | If mode is set, apply the permission recursively into the artifact if it is a folder |  |
| s3 | [S3Artifact](#s3-artifact)| `S3Artifact` |  | |  |  |
| subPath | string| `string` |  | | SubPath allows an artifact to be sourced from a subpath within the specified source |  |



### <span id="artifact-g-c"></span> ArtifactGC


> ArtifactGC describes how to delete artifacts from completed Workflows - this is embedded into the WorkflowLevelArtifactGC, and also used for individual Artifacts to override that as needed
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| podMetadata | [Metadata](#metadata)| `Metadata` |  | |  |  |
| serviceAccountName | string| `string` |  | | ServiceAccountName is an optional field for specifying the Service Account that should be assigned to the Pod doing the deletion |  |
| strategy | [ArtifactGCStrategy](#artifact-g-c-strategy)| `ArtifactGCStrategy` |  | |  |  |



### <span id="artifact-g-c-strategy"></span> ArtifactGCStrategy


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| ArtifactGCStrategy | string| string | |  |  |



### <span id="artifact-location"></span> ArtifactLocation


> It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname).
It is also used to describe the location of multiple artifacts such as the archive location
of a single workflow step, which the executor will use as a default location to store its files.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| archiveLogs | boolean| `bool` |  | | ArchiveLogs indicates if the container logs should be archived |  |
| artifactory | [ArtifactoryArtifact](#artifactory-artifact)| `ArtifactoryArtifact` |  | |  |  |
| azure | [AzureArtifact](#azure-artifact)| `AzureArtifact` |  | |  |  |
| gcs | [GCSArtifact](#g-c-s-artifact)| `GCSArtifact` |  | |  |  |
| git | [GitArtifact](#git-artifact)| `GitArtifact` |  | |  |  |
| hdfs | [HDFSArtifact](#h-d-f-s-artifact)| `HDFSArtifact` |  | |  |  |
| http | [HTTPArtifact](#http-artifact)| `HTTPArtifact` |  | |  |  |
| oss | [OSSArtifact](#o-s-s-artifact)| `OSSArtifact` |  | |  |  |
| raw | [RawArtifact](#raw-artifact)| `RawArtifact` |  | |  |  |
| s3 | [S3Artifact](#s3-artifact)| `S3Artifact` |  | |  |  |



### <span id="artifact-paths"></span> ArtifactPaths


> ArtifactPaths expands a step from a collection of artifacts
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| archive | [ArchiveStrategy](#archive-strategy)| `ArchiveStrategy` |  | |  |  |
| archiveLogs | boolean| `bool` |  | | ArchiveLogs indicates if the container logs should be archived |  |
| artifactGC | [ArtifactGC](#artifact-g-c)| `ArtifactGC` |  | |  |  |
| artifactory | [ArtifactoryArtifact](#artifactory-artifact)| `ArtifactoryArtifact` |  | |  |  |
| azure | [AzureArtifact](#azure-artifact)| `AzureArtifact` |  | |  |  |
| deleted | boolean| `bool` |  | | Has this been deleted? |  |
| from | string| `string` |  | | From allows an artifact to reference an artifact from a previous step |  |
| fromExpression | string| `string` |  | | FromExpression, if defined, is evaluated to specify the value for the artifact |  |
| gcs | [GCSArtifact](#g-c-s-artifact)| `GCSArtifact` |  | |  |  |
| git | [GitArtifact](#git-artifact)| `GitArtifact` |  | |  |  |
| globalName | string| `string` |  | | GlobalName exports an output artifact to the global scope, making it available as</br>'{{workflow.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts |  |
| hdfs | [HDFSArtifact](#h-d-f-s-artifact)| `HDFSArtifact` |  | |  |  |
| http | [HTTPArtifact](#http-artifact)| `HTTPArtifact` |  | |  |  |
| mode | int32 (formatted integer)| `int32` |  | | mode bits to use on this file, must be a value between 0 and 0777</br>set when loading input artifacts. |  |
| name | string| `string` |  | | name of the artifact. must be unique within a template's inputs/outputs. |  |
| optional | boolean| `bool` |  | | Make Artifacts optional, if Artifacts doesn't generate or exist |  |
| oss | [OSSArtifact](#o-s-s-artifact)| `OSSArtifact` |  | |  |  |
| path | string| `string` |  | | Path is the container path to the artifact |  |
| raw | [RawArtifact](#raw-artifact)| `RawArtifact` |  | |  |  |
| recurseMode | boolean| `bool` |  | | If mode is set, apply the permission recursively into the artifact if it is a folder |  |
| s3 | [S3Artifact](#s3-artifact)| `S3Artifact` |  | |  |  |
| subPath | string| `string` |  | | SubPath allows an artifact to be sourced from a subpath within the specified source |  |



### <span id="artifactory-artifact"></span> ArtifactoryArtifact


> ArtifactoryArtifact is the location of an artifactory artifact
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| passwordSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| url | string| `string` |  | | URL of the artifact |  |
| usernameSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |



### <span id="artifacts"></span> Artifacts


  

[][Artifact](#artifact)

### <span id="azure-artifact"></span> AzureArtifact


> AzureArtifact is the location of a an Azure Storage artifact
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| accountKeySecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| blob | string| `string` |  | | Blob is the blob name (i.e., path) in the container where the artifact resides |  |
| container | string| `string` |  | | Container is the container where resources will be stored |  |
| endpoint | string| `string` |  | | Endpoint is the service url associated with an account. It is most likely "https://<ACCOUNT_NAME>.blob.core.windows.net" |  |
| useSDKCreds | boolean| `bool` |  | | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. |  |



### <span id="azure-data-disk-caching-mode"></span> AzureDataDiskCachingMode


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| AzureDataDiskCachingMode | string| string | | +enum |  |



### <span id="azure-data-disk-kind"></span> AzureDataDiskKind


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| AzureDataDiskKind | string| string | | +enum |  |



### <span id="azure-disk-volume-source"></span> AzureDiskVolumeSource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| cachingMode | [AzureDataDiskCachingMode](#azure-data-disk-caching-mode)| `AzureDataDiskCachingMode` |  | |  |  |
| diskName | string| `string` |  | | diskName is the Name of the data disk in the blob storage |  |
| diskURI | string| `string` |  | | diskURI is the URI of data disk in the blob storage |  |
| fsType | string| `string` |  | | fsType is Filesystem type to mount.</br>Must be a filesystem type supported by the host operating system.</br>Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.</br>+optional</br>+default="ext4" |  |
| kind | [AzureDataDiskKind](#azure-data-disk-kind)| `AzureDataDiskKind` |  | |  |  |
| readOnly | boolean| `bool` |  | | readOnly Defaults to false (read/write). ReadOnly here will force</br>the ReadOnly setting in VolumeMounts.</br>+optional</br>+default=false |  |



### <span id="azure-file-volume-source"></span> AzureFileVolumeSource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| readOnly | boolean| `bool` |  | | readOnly defaults to false (read/write). ReadOnly here will force</br>the ReadOnly setting in VolumeMounts.</br>+optional |  |
| secretName | string| `string` |  | | secretName is the  name of secret that contains Azure Storage Account Name and Key |  |
| shareName | string| `string` |  | | shareName is the azure share Name |  |



### <span id="backoff"></span> Backoff


> Backoff is a backoff strategy to use within retryStrategy
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| duration | string| `string` |  | | Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. "2m", "1h") |  |
| factor | [IntOrString](#int-or-string)| `IntOrString` |  | |  |  |
| maxDuration | string| `string` |  | | MaxDuration is the maximum amount of time allowed for a workflow in the backoff strategy.</br>It is important to note that if the workflow template includes activeDeadlineSeconds, the pod's deadline is initially set with activeDeadlineSeconds.</br>However, when the workflow fails, the pod's deadline is then overridden by maxDuration.</br>This ensures that the workflow does not exceed the specified maximum duration when retries are involved. |  |



### <span id="basic-auth"></span> BasicAuth


> BasicAuth describes the secret selectors required for basic authentication
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| passwordSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| usernameSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |



### <span id="c-s-i-volume-source"></span> CSIVolumeSource


> Represents a source location of a volume to mount, managed by an external CSI driver
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| driver | string| `string` |  | | driver is the name of the CSI driver that handles this volume.</br>Consult with your admin for the correct name as registered in the cluster. |  |
| fsType | string| `string` |  | | fsType to mount. Ex. "ext4", "xfs", "ntfs".</br>If not provided, the empty value is passed to the associated CSI driver</br>which will determine the default filesystem to apply.</br>+optional |  |
| nodePublishSecretRef | [LocalObjectReference](#local-object-reference)| `LocalObjectReference` |  | |  |  |
| readOnly | boolean| `bool` |  | | readOnly specifies a read-only configuration for the volume.</br>Defaults to false (read/write).</br>+optional |  |
| volumeAttributes | map of string| `map[string]string` |  | | volumeAttributes stores driver-specific properties that are passed to the CSI</br>driver. Consult your driver's documentation for supported values.</br>+optional |  |



### <span id="cache"></span> Cache


> Cache is the configuration for the type of cache to be used
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| configMap | [ConfigMapKeySelector](#config-map-key-selector)| `ConfigMapKeySelector` |  | |  |  |



### <span id="capabilities"></span> Capabilities


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| add | [][Capability](#capability)| `[]Capability` |  | | Added capabilities</br>+optional</br>+listType=atomic |  |
| drop | [][Capability](#capability)| `[]Capability` |  | | Removed capabilities</br>+optional</br>+listType=atomic |  |



### <span id="capability"></span> Capability


> Capability represent POSIX capabilities type
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Capability | string| string | | Capability represent POSIX capabilities type |  |



### <span id="ceph-f-s-volume-source"></span> CephFSVolumeSource


> Represents a Ceph Filesystem mount that lasts the lifetime of a pod
Cephfs volumes do not support ownership management or SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| monitors | []string| `[]string` |  | | monitors is Required: Monitors is a collection of Ceph monitors</br>More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it</br>+listType=atomic |  |
| path | string| `string` |  | | path is Optional: Used as the mounted root, rather than the full Ceph tree, default is /</br>+optional |  |
| readOnly | boolean| `bool` |  | | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force</br>the ReadOnly setting in VolumeMounts.</br>More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it</br>+optional |  |
| secretFile | string| `string` |  | | secretFile is Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret</br>More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it</br>+optional |  |
| secretRef | [LocalObjectReference](#local-object-reference)| `LocalObjectReference` |  | |  |  |
| user | string| `string` |  | | user is optional: User is the rados user name, default is admin</br>More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it</br>+optional |  |



### <span id="cinder-volume-source"></span> CinderVolumeSource


> A Cinder volume must exist before mounting to a container.
The volume must also be in the same region as the kubelet.
Cinder volumes support ownership management and SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fsType | string| `string` |  | | fsType is the filesystem type to mount.</br>Must be a filesystem type supported by the host operating system.</br>Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.</br>More info: https://examples.k8s.io/mysql-cinder-pd/README.md</br>+optional |  |
| readOnly | boolean| `bool` |  | | readOnly defaults to false (read/write). ReadOnly here will force</br>the ReadOnly setting in VolumeMounts.</br>More info: https://examples.k8s.io/mysql-cinder-pd/README.md</br>+optional |  |
| secretRef | [LocalObjectReference](#local-object-reference)| `LocalObjectReference` |  | |  |  |
| volumeID | string| `string` |  | | volumeID used to identify the volume in cinder.</br>More info: https://examples.k8s.io/mysql-cinder-pd/README.md |  |



### <span id="client-cert-auth"></span> ClientCertAuth


> ClientCertAuth holds necessary information for client authentication via certificates
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| clientCertSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| clientKeySecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |



### <span id="cluster-trust-bundle-projection"></span> ClusterTrustBundleProjection


> ClusterTrustBundleProjection describes how to select a set of
ClusterTrustBundle objects and project their contents into the pod
filesystem.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| labelSelector | [LabelSelector](#label-selector)| `LabelSelector` |  | |  |  |
| name | string| `string` |  | | Select a single ClusterTrustBundle by object name.  Mutually-exclusive</br>with signerName and labelSelector.</br>+optional |  |
| optional | boolean| `bool` |  | | If true, don't block pod startup if the referenced ClusterTrustBundle(s)</br>aren't available.  If using name, then the named ClusterTrustBundle is</br>allowed not to exist.  If using signerName, then the combination of</br>signerName and labelSelector is allowed to match zero</br>ClusterTrustBundles.</br>+optional |  |
| path | string| `string` |  | | Relative path from the volume root to write the bundle. |  |
| signerName | string| `string` |  | | Select all ClusterTrustBundles that match this signer name.</br>Mutually-exclusive with name.  The contents of all selected</br>ClusterTrustBundles will be unified and deduplicated.</br>+optional |  |



### <span id="config-map-env-source"></span> ConfigMapEnvSource


> The contents of the target ConfigMap's Data field will represent the
key-value pairs as environment variables.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | | Name of the referent.</br>This field is effectively required, but due to backwards compatibility is</br>allowed to be empty. Instances of this type with an empty value here are</br>almost certainly wrong.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</br>+optional</br>+default=""</br>+kubebuilder:default=""</br>TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |  |
| optional | boolean| `bool` |  | | Specify whether the ConfigMap must be defined</br>+optional |  |



### <span id="config-map-key-selector"></span> ConfigMapKeySelector


> +structType=atomic
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| key | string| `string` |  | | The key to select. |  |
| name | string| `string` |  | | Name of the referent.</br>This field is effectively required, but due to backwards compatibility is</br>allowed to be empty. Instances of this type with an empty value here are</br>almost certainly wrong.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</br>+optional</br>+default=""</br>+kubebuilder:default=""</br>TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |  |
| optional | boolean| `bool` |  | | Specify whether the ConfigMap or its key must be defined</br>+optional |  |



### <span id="config-map-projection"></span> ConfigMapProjection


> The contents of the target ConfigMap's Data field will be presented in a
projected volume as files using the keys in the Data field as the file names,
unless the items element is populated with specific mappings of keys to paths.
Note that this is identical to a configmap volume source without the default
mode.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| items | [][KeyToPath](#key-to-path)| `[]*KeyToPath` |  | | items if unspecified, each key-value pair in the Data field of the referenced</br>ConfigMap will be projected into the volume as a file whose name is the</br>key and content is the value. If specified, the listed keys will be</br>projected into the specified paths, and unlisted keys will not be</br>present. If a key is specified which is not present in the ConfigMap,</br>the volume setup will error unless it is marked optional. Paths must be</br>relative and may not contain the '..' path or start with '..'.</br>+optional</br>+listType=atomic |  |
| name | string| `string` |  | | Name of the referent.</br>This field is effectively required, but due to backwards compatibility is</br>allowed to be empty. Instances of this type with an empty value here are</br>almost certainly wrong.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</br>+optional</br>+default=""</br>+kubebuilder:default=""</br>TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |  |
| optional | boolean| `bool` |  | | optional specify whether the ConfigMap or its keys must be defined</br>+optional |  |



### <span id="config-map-volume-source"></span> ConfigMapVolumeSource


> The contents of the target ConfigMap's Data field will be presented in a
volume as files using the keys in the Data field as the file names, unless
the items element is populated with specific mappings of keys to paths.
ConfigMap volumes support ownership management and SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| defaultMode | int32 (formatted integer)| `int32` |  | | defaultMode is optional: mode bits used to set permissions on created files by default.</br>Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.</br>YAML accepts both octal and decimal values, JSON requires decimal values for mode bits.</br>Defaults to 0644.</br>Directories within the path are not affected by this setting.</br>This might be in conflict with other options that affect the file</br>mode, like fsGroup, and the result can be other mode bits set.</br>+optional |  |
| items | [][KeyToPath](#key-to-path)| `[]*KeyToPath` |  | | items if unspecified, each key-value pair in the Data field of the referenced</br>ConfigMap will be projected into the volume as a file whose name is the</br>key and content is the value. If specified, the listed keys will be</br>projected into the specified paths, and unlisted keys will not be</br>present. If a key is specified which is not present in the ConfigMap,</br>the volume setup will error unless it is marked optional. Paths must be</br>relative and may not contain the '..' path or start with '..'.</br>+optional</br>+listType=atomic |  |
| name | string| `string` |  | | Name of the referent.</br>This field is effectively required, but due to backwards compatibility is</br>allowed to be empty. Instances of this type with an empty value here are</br>almost certainly wrong.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</br>+optional</br>+default=""</br>+kubebuilder:default=""</br>TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |  |
| optional | boolean| `bool` |  | | optional specify whether the ConfigMap or its keys must be defined</br>+optional |  |



### <span id="container"></span> Container


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| args | []string| `[]string` |  | | Arguments to the entrypoint.</br>The container image's CMD is used if this is not provided.</br>Variable references $(VAR_NAME) are expanded using the container's environment. If a variable</br>cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced</br>to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will</br>produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless</br>of whether the variable exists or not. Cannot be updated.</br>More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell</br>+optional</br>+listType=atomic |  |
| command | []string| `[]string` |  | | Entrypoint array. Not executed within a shell.</br>The container image's ENTRYPOINT is used if this is not provided.</br>Variable references $(VAR_NAME) are expanded using the container's environment. If a variable</br>cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced</br>to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will</br>produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless</br>of whether the variable exists or not. Cannot be updated.</br>More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell</br>+optional</br>+listType=atomic |  |
| env | [][EnvVar](#env-var)| `[]*EnvVar` |  | | List of environment variables to set in the container.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=name</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=name |  |
| envFrom | [][EnvFromSource](#env-from-source)| `[]*EnvFromSource` |  | | List of sources to populate environment variables in the container.</br>The keys defined within a source must be a C_IDENTIFIER. All invalid keys</br>will be reported as an event when the container is starting. When a key exists in multiple</br>sources, the value associated with the last source will take precedence.</br>Values defined by an Env with a duplicate key will take precedence.</br>Cannot be updated.</br>+optional</br>+listType=atomic |  |
| image | string| `string` |  | | Container image name.</br>More info: https://kubernetes.io/docs/concepts/containers/images</br>This field is optional to allow higher level config management to default or override</br>container images in workload controllers like Deployments and StatefulSets.</br>+optional |  |
| imagePullPolicy | [PullPolicy](#pull-policy)| `PullPolicy` |  | |  |  |
| lifecycle | [Lifecycle](#lifecycle)| `Lifecycle` |  | |  |  |
| livenessProbe | [Probe](#probe)| `Probe` |  | |  |  |
| name | string| `string` |  | | Name of the container specified as a DNS_LABEL.</br>Each container in a pod must have a unique name (DNS_LABEL).</br>Cannot be updated. |  |
| ports | [][ContainerPort](#container-port)| `[]*ContainerPort` |  | | List of ports to expose from the container. Not specifying a port here</br>DOES NOT prevent that port from being exposed. Any port which is</br>listening on the default "0.0.0.0" address inside a container will be</br>accessible from the network.</br>Modifying this array with strategic merge patch may corrupt the data.</br>For more information See https://github.com/kubernetes/kubernetes/issues/108255.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=containerPort</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=containerPort</br>+listMapKey=protocol |  |
| readinessProbe | [Probe](#probe)| `Probe` |  | |  |  |
| resizePolicy | [][ContainerResizePolicy](#container-resize-policy)| `[]*ContainerResizePolicy` |  | | Resources resize policy for the container.</br>+featureGate=InPlacePodVerticalScaling</br>+optional</br>+listType=atomic |  |
| resources | [ResourceRequirements](#resource-requirements)| `ResourceRequirements` |  | |  |  |
| restartPolicy | [ContainerRestartPolicy](#container-restart-policy)| `ContainerRestartPolicy` |  | |  |  |
| securityContext | [SecurityContext](#security-context)| `SecurityContext` |  | |  |  |
| startupProbe | [Probe](#probe)| `Probe` |  | |  |  |
| stdin | boolean| `bool` |  | | Whether this container should allocate a buffer for stdin in the container runtime. If this</br>is not set, reads from stdin in the container will always result in EOF.</br>Default is false.</br>+optional |  |
| stdinOnce | boolean| `bool` |  | | Whether the container runtime should close the stdin channel after it has been opened by</br>a single attach. When stdin is true the stdin stream will remain open across multiple attach</br>sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the</br>first client attaches to stdin, and then remains open and accepts data until the client disconnects,</br>at which time stdin is closed and remains closed until the container is restarted. If this</br>flag is false, a container processes that reads from stdin will never receive an EOF.</br>Default is false</br>+optional |  |
| terminationMessagePath | string| `string` |  | | Optional: Path at which the file to which the container's termination message</br>will be written is mounted into the container's filesystem.</br>Message written is intended to be brief final status, such as an assertion failure message.</br>Will be truncated by the node if greater than 4096 bytes. The total message length across</br>all containers will be limited to 12kb.</br>Defaults to /dev/termination-log.</br>Cannot be updated.</br>+optional |  |
| terminationMessagePolicy | [TerminationMessagePolicy](#termination-message-policy)| `TerminationMessagePolicy` |  | |  |  |
| tty | boolean| `bool` |  | | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true.</br>Default is false.</br>+optional |  |
| volumeDevices | [][VolumeDevice](#volume-device)| `[]*VolumeDevice` |  | | volumeDevices is the list of block devices to be used by the container.</br>+patchMergeKey=devicePath</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=devicePath</br>+optional |  |
| volumeMounts | [][VolumeMount](#volume-mount)| `[]*VolumeMount` |  | | Pod volumes to mount into the container's filesystem.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=mountPath</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=mountPath |  |
| workingDir | string| `string` |  | | Container's working directory.</br>If not specified, the container runtime's default will be used, which</br>might be configured in the container image.</br>Cannot be updated.</br>+optional |  |



### <span id="container-node"></span> ContainerNode


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| args | []string| `[]string` |  | | Arguments to the entrypoint.</br>The container image's CMD is used if this is not provided.</br>Variable references $(VAR_NAME) are expanded using the container's environment. If a variable</br>cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced</br>to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will</br>produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless</br>of whether the variable exists or not. Cannot be updated.</br>More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell</br>+optional</br>+listType=atomic |  |
| command | []string| `[]string` |  | | Entrypoint array. Not executed within a shell.</br>The container image's ENTRYPOINT is used if this is not provided.</br>Variable references $(VAR_NAME) are expanded using the container's environment. If a variable</br>cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced</br>to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will</br>produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless</br>of whether the variable exists or not. Cannot be updated.</br>More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell</br>+optional</br>+listType=atomic |  |
| dependencies | []string| `[]string` |  | |  |  |
| env | [][EnvVar](#env-var)| `[]*EnvVar` |  | | List of environment variables to set in the container.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=name</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=name |  |
| envFrom | [][EnvFromSource](#env-from-source)| `[]*EnvFromSource` |  | | List of sources to populate environment variables in the container.</br>The keys defined within a source must be a C_IDENTIFIER. All invalid keys</br>will be reported as an event when the container is starting. When a key exists in multiple</br>sources, the value associated with the last source will take precedence.</br>Values defined by an Env with a duplicate key will take precedence.</br>Cannot be updated.</br>+optional</br>+listType=atomic |  |
| image | string| `string` |  | | Container image name.</br>More info: https://kubernetes.io/docs/concepts/containers/images</br>This field is optional to allow higher level config management to default or override</br>container images in workload controllers like Deployments and StatefulSets.</br>+optional |  |
| imagePullPolicy | [PullPolicy](#pull-policy)| `PullPolicy` |  | |  |  |
| lifecycle | [Lifecycle](#lifecycle)| `Lifecycle` |  | |  |  |
| livenessProbe | [Probe](#probe)| `Probe` |  | |  |  |
| name | string| `string` |  | | Name of the container specified as a DNS_LABEL.</br>Each container in a pod must have a unique name (DNS_LABEL).</br>Cannot be updated. |  |
| ports | [][ContainerPort](#container-port)| `[]*ContainerPort` |  | | List of ports to expose from the container. Not specifying a port here</br>DOES NOT prevent that port from being exposed. Any port which is</br>listening on the default "0.0.0.0" address inside a container will be</br>accessible from the network.</br>Modifying this array with strategic merge patch may corrupt the data.</br>For more information See https://github.com/kubernetes/kubernetes/issues/108255.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=containerPort</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=containerPort</br>+listMapKey=protocol |  |
| readinessProbe | [Probe](#probe)| `Probe` |  | |  |  |
| resizePolicy | [][ContainerResizePolicy](#container-resize-policy)| `[]*ContainerResizePolicy` |  | | Resources resize policy for the container.</br>+featureGate=InPlacePodVerticalScaling</br>+optional</br>+listType=atomic |  |
| resources | [ResourceRequirements](#resource-requirements)| `ResourceRequirements` |  | |  |  |
| restartPolicy | [ContainerRestartPolicy](#container-restart-policy)| `ContainerRestartPolicy` |  | |  |  |
| securityContext | [SecurityContext](#security-context)| `SecurityContext` |  | |  |  |
| startupProbe | [Probe](#probe)| `Probe` |  | |  |  |
| stdin | boolean| `bool` |  | | Whether this container should allocate a buffer for stdin in the container runtime. If this</br>is not set, reads from stdin in the container will always result in EOF.</br>Default is false.</br>+optional |  |
| stdinOnce | boolean| `bool` |  | | Whether the container runtime should close the stdin channel after it has been opened by</br>a single attach. When stdin is true the stdin stream will remain open across multiple attach</br>sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the</br>first client attaches to stdin, and then remains open and accepts data until the client disconnects,</br>at which time stdin is closed and remains closed until the container is restarted. If this</br>flag is false, a container processes that reads from stdin will never receive an EOF.</br>Default is false</br>+optional |  |
| terminationMessagePath | string| `string` |  | | Optional: Path at which the file to which the container's termination message</br>will be written is mounted into the container's filesystem.</br>Message written is intended to be brief final status, such as an assertion failure message.</br>Will be truncated by the node if greater than 4096 bytes. The total message length across</br>all containers will be limited to 12kb.</br>Defaults to /dev/termination-log.</br>Cannot be updated.</br>+optional |  |
| terminationMessagePolicy | [TerminationMessagePolicy](#termination-message-policy)| `TerminationMessagePolicy` |  | |  |  |
| tty | boolean| `bool` |  | | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true.</br>Default is false.</br>+optional |  |
| volumeDevices | [][VolumeDevice](#volume-device)| `[]*VolumeDevice` |  | | volumeDevices is the list of block devices to be used by the container.</br>+patchMergeKey=devicePath</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=devicePath</br>+optional |  |
| volumeMounts | [][VolumeMount](#volume-mount)| `[]*VolumeMount` |  | | Pod volumes to mount into the container's filesystem.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=mountPath</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=mountPath |  |
| workingDir | string| `string` |  | | Container's working directory.</br>If not specified, the container runtime's default will be used, which</br>might be configured in the container image.</br>Cannot be updated.</br>+optional |  |



### <span id="container-port"></span> ContainerPort


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| containerPort | int32 (formatted integer)| `int32` |  | | Number of port to expose on the pod's IP address.</br>This must be a valid port number, 0 < x < 65536. |  |
| hostIP | string| `string` |  | | What host IP to bind the external port to.</br>+optional |  |
| hostPort | int32 (formatted integer)| `int32` |  | | Number of port to expose on the host.</br>If specified, this must be a valid port number, 0 < x < 65536.</br>If HostNetwork is specified, this must match ContainerPort.</br>Most containers do not need this.</br>+optional |  |
| name | string| `string` |  | | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each</br>named port in a pod must have a unique name. Name for the port that can be</br>referred to by services.</br>+optional |  |
| protocol | [Protocol](#protocol)| `Protocol` |  | |  |  |



### <span id="container-resize-policy"></span> ContainerResizePolicy


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| resourceName | [ResourceName](#resource-name)| `ResourceName` |  | |  |  |
| restartPolicy | [ResourceResizeRestartPolicy](#resource-resize-restart-policy)| `ResourceResizeRestartPolicy` |  | |  |  |



### <span id="container-restart-policy"></span> ContainerRestartPolicy


> This may only be set for init containers and only allowed value is "Always".
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| ContainerRestartPolicy | string| string | | This may only be set for init containers and only allowed value is "Always". |  |



### <span id="container-set-retry-strategy"></span> ContainerSetRetryStrategy


> ContainerSetRetryStrategy provides controls on how to retry a container set
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| duration | string| `string` |  | | Duration is the time between each retry, examples values are "300ms", "1s" or "5m".</br>Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". |  |
| retries | [IntOrString](#int-or-string)| `IntOrString` |  | |  |  |



### <span id="container-set-template"></span> ContainerSetTemplate


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| containers | [][ContainerNode](#container-node)| `[]*ContainerNode` |  | |  |  |
| retryStrategy | [ContainerSetRetryStrategy](#container-set-retry-strategy)| `ContainerSetRetryStrategy` |  | |  |  |
| volumeMounts | [][VolumeMount](#volume-mount)| `[]*VolumeMount` |  | |  |  |



### <span id="continue-on"></span> ContinueOn


> It can be specified if the workflow should continue when the pod errors, fails or both.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| error | boolean| `bool` |  | | +optional |  |
| failed | boolean| `bool` |  | | +optional |  |



### <span id="counter"></span> Counter


> Counter is a Counter prometheus metric
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| value | string| `string` |  | | Value is the value of the metric |  |



### <span id="create-s3-bucket-options"></span> CreateS3BucketOptions


> CreateS3BucketOptions options used to determine automatic automatic bucket-creation process
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| objectLocking | boolean| `bool` |  | | ObjectLocking Enable object locking |  |



### <span id="d-a-g-task"></span> DAGTask


> DAGTask represents a node in the graph during DAG execution
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| arguments | [Arguments](#arguments)| `Arguments` |  | |  |  |
| continueOn | [ContinueOn](#continue-on)| `ContinueOn` |  | |  |  |
| dependencies | []string| `[]string` |  | | Dependencies are name of other targets which this depends on |  |
| depends | string| `string` |  | | Depends are name of other targets which this depends on |  |
| hooks | [LifecycleHooks](#lifecycle-hooks)| `LifecycleHooks` |  | |  |  |
| inline | [Template](#template)| `Template` |  | |  |  |
| name | string| `string` |  | | Name is the name of the target |  |
| onExit | string| `string` |  | | OnExit is a template reference which is invoked at the end of the</br>template, irrespective of the success, failure, or error of the</br>primary template.</br>DEPRECATED: Use Hooks[exit].Template instead. |  |
| template | string| `string` |  | | Name of template to execute |  |
| templateRef | [TemplateRef](#template-ref)| `TemplateRef` |  | |  |  |
| when | string| `string` |  | | When is an expression in which the task should conditionally execute |  |
| withItems | [][Item](#item)| `[]Item` |  | | WithItems expands a task into multiple parallel tasks from the items in the list |  |
| withParam | string| `string` |  | | WithParam expands a task into multiple parallel tasks from the value in the parameter,</br>which is expected to be a JSON list. |  |
| withSequence | [Sequence](#sequence)| `Sequence` |  | |  |  |



### <span id="d-a-g-template"></span> DAGTemplate


> DAGTemplate is a template subtype for directed acyclic graph templates
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| failFast | boolean| `bool` |  | | This flag is for DAG logic. The DAG logic has a built-in "fail fast" feature to stop scheduling new steps,</br>as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completed</br>before failing the DAG itself.</br>The FailFast flag default is true,  if set to false, it will allow a DAG to run all branches of the DAG to</br>completion (either success or failure), regardless of the failed outcomes of branches in the DAG.</br>More info and example about this feature at https://github.com/argoproj/argo-workflows/issues/1442 |  |
| target | string| `string` |  | | Target are one or more names of targets to execute in a DAG |  |
| tasks | [][DAGTask](#d-a-g-task)| `[]*DAGTask` |  | | Tasks are a list of DAG tasks</br>+patchStrategy=merge</br>+patchMergeKey=name |  |



### <span id="data"></span> Data


> Data is a data template
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| source | [DataSource](#data-source)| `DataSource` |  | |  |  |
| transformation | [Transformation](#transformation)| `Transformation` |  | |  |  |



### <span id="data-source"></span> DataSource


> DataSource sources external data into a data template
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| artifactPaths | [ArtifactPaths](#artifact-paths)| `ArtifactPaths` |  | |  |  |



### <span id="downward-api-projection"></span> DownwardAPIProjection


> Note that this is identical to a downwardAPI volume source without the default
mode.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| items | [][DownwardAPIVolumeFile](#downward-api-volume-file)| `[]*DownwardAPIVolumeFile` |  | | Items is a list of DownwardAPIVolume file</br>+optional</br>+listType=atomic |  |



### <span id="downward-api-volume-file"></span> DownwardAPIVolumeFile


> DownwardAPIVolumeFile represents information to create the file containing the pod field
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fieldRef | [ObjectFieldSelector](#object-field-selector)| `ObjectFieldSelector` |  | |  |  |
| mode | int32 (formatted integer)| `int32` |  | | Optional: mode bits used to set permissions on this file, must be an octal value</br>between 0000 and 0777 or a decimal value between 0 and 511.</br>YAML accepts both octal and decimal values, JSON requires decimal values for mode bits.</br>If not specified, the volume defaultMode will be used.</br>This might be in conflict with other options that affect the file</br>mode, like fsGroup, and the result can be other mode bits set.</br>+optional |  |
| path | string| `string` |  | | Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..' |  |
| resourceFieldRef | [ResourceFieldSelector](#resource-field-selector)| `ResourceFieldSelector` |  | |  |  |



### <span id="downward-api-volume-source"></span> DownwardAPIVolumeSource


> Downward API volumes support ownership management and SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| defaultMode | int32 (formatted integer)| `int32` |  | | Optional: mode bits to use on created files by default. Must be a</br>Optional: mode bits used to set permissions on created files by default.</br>Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.</br>YAML accepts both octal and decimal values, JSON requires decimal values for mode bits.</br>Defaults to 0644.</br>Directories within the path are not affected by this setting.</br>This might be in conflict with other options that affect the file</br>mode, like fsGroup, and the result can be other mode bits set.</br>+optional |  |
| items | [][DownwardAPIVolumeFile](#downward-api-volume-file)| `[]*DownwardAPIVolumeFile` |  | | Items is a list of downward API volume file</br>+optional</br>+listType=atomic |  |



### <span id="duration"></span> Duration


> Duration is a wrapper around time.Duration which supports correct
marshaling to YAML and JSON. In particular, it marshals into strings, which
can be used as map keys in json.
  



[interface{}](#interface)

### <span id="empty-dir-volume-source"></span> EmptyDirVolumeSource


> Empty directory volumes support ownership management and SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| medium | [StorageMedium](#storage-medium)| `StorageMedium` |  | |  |  |
| sizeLimit | [Quantity](#quantity)| `Quantity` |  | |  |  |



### <span id="env-from-source"></span> EnvFromSource


> EnvFromSource represents the source of a set of ConfigMaps
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| configMapRef | [ConfigMapEnvSource](#config-map-env-source)| `ConfigMapEnvSource` |  | |  |  |
| prefix | string| `string` |  | | An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.</br>+optional |  |
| secretRef | [SecretEnvSource](#secret-env-source)| `SecretEnvSource` |  | |  |  |



### <span id="env-var"></span> EnvVar


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | | Name of the environment variable. Must be a C_IDENTIFIER. |  |
| value | string| `string` |  | | Variable references $(VAR_NAME) are expanded</br>using the previously defined environment variables in the container and</br>any service environment variables. If a variable cannot be resolved,</br>the reference in the input string will be unchanged. Double $$ are reduced</br>to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e.</br>"$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)".</br>Escaped references will never be expanded, regardless of whether the variable</br>exists or not.</br>Defaults to "".</br>+optional |  |
| valueFrom | [EnvVarSource](#env-var-source)| `EnvVarSource` |  | |  |  |



### <span id="env-var-source"></span> EnvVarSource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| configMapKeyRef | [ConfigMapKeySelector](#config-map-key-selector)| `ConfigMapKeySelector` |  | |  |  |
| fieldRef | [ObjectFieldSelector](#object-field-selector)| `ObjectFieldSelector` |  | |  |  |
| resourceFieldRef | [ResourceFieldSelector](#resource-field-selector)| `ResourceFieldSelector` |  | |  |  |
| secretKeyRef | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |



### <span id="ephemeral-volume-source"></span> EphemeralVolumeSource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| volumeClaimTemplate | [PersistentVolumeClaimTemplate](#persistent-volume-claim-template)| `PersistentVolumeClaimTemplate` |  | |  |  |



### <span id="exec-action"></span> ExecAction


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| command | []string| `[]string` |  | | Command is the command line to execute inside the container, the working directory for the</br>command  is root ('/') in the container's filesystem. The command is simply exec'd, it is</br>not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use</br>a shell, you need to explicitly call out to that shell.</br>Exit status of 0 is treated as live/healthy and non-zero is unhealthy.</br>+optional</br>+listType=atomic |  |



### <span id="execute-template-args"></span> ExecuteTemplateArgs


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| template | [Template](#template)| `Template` | ✓ | |  |  |
| workflow | [Workflow](#workflow)| `Workflow` | ✓ | |  |  |



### <span id="execute-template-reply"></span> ExecuteTemplateReply


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| node | [NodeResult](#node-result)| `NodeResult` |  | |  |  |
| requeue | [Duration](#duration)| `Duration` |  | |  |  |



### <span id="executor-config"></span> ExecutorConfig


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| serviceAccountName | string| `string` |  | | ServiceAccountName specifies the service account name of the executor container. |  |



### <span id="f-c-volume-source"></span> FCVolumeSource


> Fibre Channel volumes can only be mounted as read/write once.
Fibre Channel volumes support ownership management and SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fsType | string| `string` |  | | fsType is the filesystem type to mount.</br>Must be a filesystem type supported by the host operating system.</br>Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.</br>TODO: how do we prevent errors in the filesystem from compromising the machine</br>+optional |  |
| lun | int32 (formatted integer)| `int32` |  | | lun is Optional: FC target lun number</br>+optional |  |
| readOnly | boolean| `bool` |  | | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force</br>the ReadOnly setting in VolumeMounts.</br>+optional |  |
| targetWWNs | []string| `[]string` |  | | targetWWNs is Optional: FC target worldwide names (WWNs)</br>+optional</br>+listType=atomic |  |
| wwids | []string| `[]string` |  | | wwids Optional: FC volume world wide identifiers (wwids)</br>Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously.</br>+optional</br>+listType=atomic |  |



### <span id="fields-v1"></span> FieldsV1


> Each key is either a '.' representing the field itself, and will always map to an empty set,
or a string representing a sub-field or item. The string will follow one of these four formats:
'f:<name>', where <name> is the name of a field in a struct, or key in a map
'v:<value>', where <value> is the exact json formatted value of a list item
'i:<index>', where <index> is position of a item in a list
'k:<keys>', where <keys> is a map of  a list item's key fields to their unique values
If a key maps to an empty Fields value, the field that key represents is part of the set.

The exact format is defined in sigs.k8s.io/structured-merge-diff
+protobuf.options.(gogoproto.goproto_stringer)=false
  



[interface{}](#interface)

### <span id="flex-volume-source"></span> FlexVolumeSource


> FlexVolume represents a generic volume resource that is
provisioned/attached using an exec based plugin.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| driver | string| `string` |  | | driver is the name of the driver to use for this volume. |  |
| fsType | string| `string` |  | | fsType is the filesystem type to mount.</br>Must be a filesystem type supported by the host operating system.</br>Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script.</br>+optional |  |
| options | map of string| `map[string]string` |  | | options is Optional: this field holds extra command options if any.</br>+optional |  |
| readOnly | boolean| `bool` |  | | readOnly is Optional: defaults to false (read/write). ReadOnly here will force</br>the ReadOnly setting in VolumeMounts.</br>+optional |  |
| secretRef | [LocalObjectReference](#local-object-reference)| `LocalObjectReference` |  | |  |  |



### <span id="flocker-volume-source"></span> FlockerVolumeSource


> One and only one of datasetName and datasetUUID should be set.
Flocker volumes do not support ownership management or SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| datasetName | string| `string` |  | | datasetName is Name of the dataset stored as metadata -> name on the dataset for Flocker</br>should be considered as deprecated</br>+optional |  |
| datasetUUID | string| `string` |  | | datasetUUID is the UUID of the dataset. This is unique identifier of a Flocker dataset</br>+optional |  |



### <span id="g-c-e-persistent-disk-volume-source"></span> GCEPersistentDiskVolumeSource


> A GCE PD must exist before mounting to a container. The disk must
also be in the same GCE project and zone as the kubelet. A GCE PD
can only be mounted as read/write once or read-only many times. GCE
PDs support ownership management and SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fsType | string| `string` |  | | fsType is filesystem type of the volume that you want to mount.</br>Tip: Ensure that the filesystem type is supported by the host operating system.</br>Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk</br>TODO: how do we prevent errors in the filesystem from compromising the machine</br>+optional |  |
| partition | int32 (formatted integer)| `int32` |  | | partition is the partition in the volume that you want to mount.</br>If omitted, the default is to mount by volume name.</br>Examples: For volume /dev/sda1, you specify the partition as "1".</br>Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty).</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk</br>+optional |  |
| pdName | string| `string` |  | | pdName is unique name of the PD resource in GCE. Used to identify the disk in GCE.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk |  |
| readOnly | boolean| `bool` |  | | readOnly here will force the ReadOnly setting in VolumeMounts.</br>Defaults to false.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk</br>+optional |  |



### <span id="g-c-s-artifact"></span> GCSArtifact


> GCSArtifact is the location of a GCS artifact
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| bucket | string| `string` |  | | Bucket is the name of the bucket |  |
| key | string| `string` |  | | Key is the path in the bucket where the artifact resides |  |
| serviceAccountKeySecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |



### <span id="g-rpc-action"></span> GRPCAction


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| port | int32 (formatted integer)| `int32` |  | | Port number of the gRPC service. Number must be in the range 1 to 65535. |  |
| service | string| `string` |  | | Service is the name of the service to place in the gRPC HealthCheckRequest</br>(see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).</br></br>If this is not specified, the default behavior is defined by gRPC.</br>+optional</br>+default="" |  |



### <span id="gauge"></span> Gauge


> Gauge is a Gauge prometheus metric
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| operation | [GaugeOperation](#gauge-operation)| `GaugeOperation` |  | |  |  |
| realtime | boolean| `bool` |  | | Realtime emits this metric in real time if applicable |  |
| value | string| `string` |  | | Value is the value to be used in the operation with the metric's current value. If no operation is set,</br>value is the value of the metric |  |



### <span id="gauge-operation"></span> GaugeOperation


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| GaugeOperation | string| string | |  |  |



### <span id="git-artifact"></span> GitArtifact


> GitArtifact is the location of an git artifact
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| branch | string| `string` |  | | Branch is the branch to fetch when `SingleBranch` is enabled |  |
| depth | uint64 (formatted integer)| `uint64` |  | | Depth specifies clones/fetches should be shallow and include the given</br>number of commits from the branch tip |  |
| disableSubmodules | boolean| `bool` |  | | DisableSubmodules disables submodules during git clone |  |
| fetch | []string| `[]string` |  | | Fetch specifies a number of refs that should be fetched before checkout |  |
| insecureIgnoreHostKey | boolean| `bool` |  | | InsecureIgnoreHostKey disables SSH strict host key checking during git clone |  |
| insecureSkipTLS | boolean| `bool` |  | | InsecureSkipTLS disables server certificate verification resulting in insecure HTTPS connections |  |
| passwordSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| repo | string| `string` |  | | Repo is the git repository |  |
| revision | string| `string` |  | | Revision is the git commit, tag, branch to checkout |  |
| singleBranch | boolean| `bool` |  | | SingleBranch enables single branch clone, using the `branch` parameter |  |
| sshPrivateKeySecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| usernameSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |



### <span id="git-repo-volume-source"></span> GitRepoVolumeSource


> DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount an
EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir
into the Pod's container.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| directory | string| `string` |  | | directory is the target directory name.</br>Must not contain or start with '..'.  If '.' is supplied, the volume directory will be the</br>git repository.  Otherwise, if specified, the volume will contain the git repository in</br>the subdirectory with the given name.</br>+optional |  |
| repository | string| `string` |  | | repository is the URL |  |
| revision | string| `string` |  | | revision is the commit hash for the specified revision.</br>+optional |  |



### <span id="glusterfs-volume-source"></span> GlusterfsVolumeSource


> Glusterfs volumes do not support ownership management or SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| endpoints | string| `string` |  | | endpoints is the endpoint name that details Glusterfs topology.</br>More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod |  |
| path | string| `string` |  | | path is the Glusterfs volume path.</br>More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod |  |
| readOnly | boolean| `bool` |  | | readOnly here will force the Glusterfs volume to be mounted with read-only permissions.</br>Defaults to false.</br>More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod</br>+optional |  |



### <span id="h-d-f-s-artifact"></span> HDFSArtifact


> HDFSArtifact is the location of an HDFS artifact
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| addresses | []string| `[]string` |  | | Addresses is accessible addresses of HDFS name nodes |  |
| dataTransferProtection | string| `string` |  | | DataTransferProtection is the protection level for HDFS data transfer.</br>It corresponds to the dfs.data.transfer.protection configuration in HDFS. |  |
| force | boolean| `bool` |  | | Force copies a file forcibly even if it exists |  |
| hdfsUser | string| `string` |  | | HDFSUser is the user to access HDFS file system.</br>It is ignored if either ccache or keytab is used. |  |
| krbCCacheSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| krbConfigConfigMap | [ConfigMapKeySelector](#config-map-key-selector)| `ConfigMapKeySelector` |  | |  |  |
| krbKeytabSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| krbRealm | string| `string` |  | | KrbRealm is the Kerberos realm used with Kerberos keytab</br>It must be set if keytab is used. |  |
| krbServicePrincipalName | string| `string` |  | | KrbServicePrincipalName is the principal name of Kerberos service</br>It must be set if either ccache or keytab is used. |  |
| krbUsername | string| `string` |  | | KrbUsername is the Kerberos username used with Kerberos keytab</br>It must be set if keytab is used. |  |
| path | string| `string` |  | | Path is a file path in HDFS |  |



### <span id="http"></span> HTTP


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| body | string| `string` |  | | Body is content of the HTTP Request |  |
| bodyFrom | [HTTPBodySource](#http-body-source)| `HTTPBodySource` |  | |  |  |
| headers | [HTTPHeaders](#http-headers)| `HTTPHeaders` |  | |  |  |
| insecureSkipVerify | boolean| `bool` |  | | InsecureSkipVerify is a bool when if set to true will skip TLS verification for the HTTP client |  |
| method | string| `string` |  | | Method is HTTP methods for HTTP Request |  |
| successCondition | string| `string` |  | | SuccessCondition is an expression if evaluated to true is considered successful |  |
| timeoutSeconds | int64 (formatted integer)| `int64` |  | | TimeoutSeconds is request timeout for HTTP Request. Default is 30 seconds |  |
| url | string| `string` |  | | URL of the HTTP Request |  |



### <span id="http-artifact"></span> HTTPArtifact


> HTTPArtifact allows a file served on HTTP to be placed as an input artifact in a container
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| auth | [HTTPAuth](#http-auth)| `HTTPAuth` |  | |  |  |
| headers | [][Header](#header)| `[]*Header` |  | | Headers are an optional list of headers to send with HTTP requests for artifacts |  |
| url | string| `string` |  | | URL of the artifact |  |



### <span id="http-auth"></span> HTTPAuth


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| basicAuth | [BasicAuth](#basic-auth)| `BasicAuth` |  | |  |  |
| clientCert | [ClientCertAuth](#client-cert-auth)| `ClientCertAuth` |  | |  |  |
| oauth2 | [OAuth2Auth](#o-auth2-auth)| `OAuth2Auth` |  | |  |  |



### <span id="http-body-source"></span> HTTPBodySource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| bytes | []uint8 (formatted integer)| `[]uint8` |  | |  |  |



### <span id="http-get-action"></span> HTTPGetAction


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| host | string| `string` |  | | Host name to connect to, defaults to the pod IP. You probably want to set</br>"Host" in httpHeaders instead.</br>+optional |  |
| httpHeaders | [][HTTPHeader](#http-header)| `[]*HTTPHeader` |  | | Custom headers to set in the request. HTTP allows repeated headers.</br>+optional</br>+listType=atomic |  |
| path | string| `string` |  | | Path to access on the HTTP server.</br>+optional |  |
| port | [IntOrString](#int-or-string)| `IntOrString` |  | |  |  |
| scheme | [URIScheme](#uri-scheme)| `URIScheme` |  | |  |  |



### <span id="http-header"></span> HTTPHeader


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | |  |  |
| value | string| `string` |  | |  |  |
| valueFrom | [HTTPHeaderSource](#http-header-source)| `HTTPHeaderSource` |  | |  |  |



### <span id="http-header-source"></span> HTTPHeaderSource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| secretKeyRef | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |



### <span id="http-headers"></span> HTTPHeaders


  

[][HTTPHeader](#http-header)

### <span id="header"></span> Header


> Header indicate a key-value request header to be used when fetching artifacts over HTTP
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | | Name is the header name |  |
| value | string| `string` |  | | Value is the literal value to use for the header |  |



### <span id="histogram"></span> Histogram


> Histogram is a Histogram prometheus metric
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| buckets | [][Amount](#amount)| `[]Amount` |  | | Buckets is a list of bucket divisors for the histogram |  |
| value | string| `string` |  | | Value is the value of the metric |  |



### <span id="host-alias"></span> HostAlias


> HostAlias holds the mapping between IP and hostnames that will be injected as an entry in the
pod's hosts file.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| hostnames | []string| `[]string` |  | | Hostnames for the above IP address.</br>+listType=atomic |  |
| ip | string| `string` |  | | IP address of the host file entry.</br>+required |  |



### <span id="host-path-type"></span> HostPathType


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| HostPathType | string| string | | +enum |  |



### <span id="host-path-volume-source"></span> HostPathVolumeSource


> Host path volumes do not support ownership management or SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| path | string| `string` |  | | path of the directory on the host.</br>If the path is a symlink, it will follow the link to the real path.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath |  |
| type | [HostPathType](#host-path-type)| `HostPathType` |  | |  |  |



### <span id="i-s-c-s-i-volume-source"></span> ISCSIVolumeSource


> ISCSI volumes can only be mounted as read/write once.
ISCSI volumes support ownership management and SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| chapAuthDiscovery | boolean| `bool` |  | | chapAuthDiscovery defines whether support iSCSI Discovery CHAP authentication</br>+optional |  |
| chapAuthSession | boolean| `bool` |  | | chapAuthSession defines whether support iSCSI Session CHAP authentication</br>+optional |  |
| fsType | string| `string` |  | | fsType is the filesystem type of the volume that you want to mount.</br>Tip: Ensure that the filesystem type is supported by the host operating system.</br>Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi</br>TODO: how do we prevent errors in the filesystem from compromising the machine</br>+optional |  |
| initiatorName | string| `string` |  | | initiatorName is the custom iSCSI Initiator Name.</br>If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface</br><target portal>:<volume name> will be created for the connection.</br>+optional |  |
| iqn | string| `string` |  | | iqn is the target iSCSI Qualified Name. |  |
| iscsiInterface | string| `string` |  | | iscsiInterface is the interface Name that uses an iSCSI transport.</br>Defaults to 'default' (tcp).</br>+optional</br>+default="default" |  |
| lun | int32 (formatted integer)| `int32` |  | | lun represents iSCSI Target Lun number. |  |
| portals | []string| `[]string` |  | | portals is the iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port</br>is other than default (typically TCP ports 860 and 3260).</br>+optional</br>+listType=atomic |  |
| readOnly | boolean| `bool` |  | | readOnly here will force the ReadOnly setting in VolumeMounts.</br>Defaults to false.</br>+optional |  |
| secretRef | [LocalObjectReference](#local-object-reference)| `LocalObjectReference` |  | |  |  |
| targetPortal | string| `string` |  | | targetPortal is iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port</br>is other than default (typically TCP ports 860 and 3260). |  |



### <span id="image-volume-source"></span> ImageVolumeSource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| pullPolicy | [PullPolicy](#pull-policy)| `PullPolicy` |  | |  |  |
| reference | string| `string` |  | | Required: Image or artifact reference to be used.</br>Behaves in the same way as pod.spec.containers[*].image.</br>Pull secrets will be assembled in the same way as for the container image by looking up node credentials, SA image pull secrets, and pod spec image pull secrets.</br>More info: https://kubernetes.io/docs/concepts/containers/images</br>This field is optional to allow higher level config management to default or override</br>container images in workload controllers like Deployments and StatefulSets.</br>+optional |  |



### <span id="inputs"></span> Inputs


> Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| artifacts | [Artifacts](#artifacts)| `Artifacts` |  | |  |  |
| parameters | [][Parameter](#parameter)| `[]*Parameter` |  | | Parameters are a list of parameters passed as inputs</br>+patchStrategy=merge</br>+patchMergeKey=name |  |



### <span id="int-or-string"></span> IntOrString


> +protobuf=true
+protobuf.options.(gogoproto.goproto_stringer)=false
+k8s:openapi-gen=true
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| IntVal | int32 (formatted integer)| `int32` |  | |  |  |
| StrVal | string| `string` |  | |  |  |
| Type | [Type](#type)| `Type` |  | |  |  |



### <span id="item"></span> Item


> +protobuf.options.(gogoproto.goproto_stringer)=false
+kubebuilder:validation:Type=object
  



[interface{}](#interface)

### <span id="key-to-path"></span> KeyToPath


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| key | string| `string` |  | | key is the key to project. |  |
| mode | int32 (formatted integer)| `int32` |  | | mode is Optional: mode bits used to set permissions on this file.</br>Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.</br>YAML accepts both octal and decimal values, JSON requires decimal values for mode bits.</br>If not specified, the volume defaultMode will be used.</br>This might be in conflict with other options that affect the file</br>mode, like fsGroup, and the result can be other mode bits set.</br>+optional |  |
| path | string| `string` |  | | path is the relative path of the file to map the key to.</br>May not be an absolute path.</br>May not contain the path element '..'.</br>May not start with the string '..'. |  |



### <span id="label-selector"></span> LabelSelector


> A label selector is a label query over a set of resources. The result of matchLabels and
matchExpressions are ANDed. An empty label selector matches all objects. A null
label selector matches no objects.
+structType=atomic
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| matchExpressions | [][LabelSelectorRequirement](#label-selector-requirement)| `[]*LabelSelectorRequirement` |  | | matchExpressions is a list of label selector requirements. The requirements are ANDed.</br>+optional</br>+listType=atomic |  |
| matchLabels | map of string| `map[string]string` |  | | matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels</br>map is equivalent to an element of matchExpressions, whose key field is "key", the</br>operator is "In", and the values array contains only "value". The requirements are ANDed.</br>+optional |  |



### <span id="label-selector-operator"></span> LabelSelectorOperator


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| LabelSelectorOperator | string| string | |  |  |



### <span id="label-selector-requirement"></span> LabelSelectorRequirement


> A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| key | string| `string` |  | | key is the label key that the selector applies to. |  |
| operator | [LabelSelectorOperator](#label-selector-operator)| `LabelSelectorOperator` |  | |  |  |
| values | []string| `[]string` |  | | values is an array of string values. If the operator is In or NotIn,</br>the values array must be non-empty. If the operator is Exists or DoesNotExist,</br>the values array must be empty. This array is replaced during a strategic</br>merge patch.</br>+optional</br>+listType=atomic |  |



### <span id="lifecycle"></span> Lifecycle


> Lifecycle describes actions that the management system should take in response to container lifecycle
events. For the PostStart and PreStop lifecycle handlers, management of the container blocks
until the action is complete, unless the container process fails, in which case the handler is aborted.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| postStart | [LifecycleHandler](#lifecycle-handler)| `LifecycleHandler` |  | |  |  |
| preStop | [LifecycleHandler](#lifecycle-handler)| `LifecycleHandler` |  | |  |  |



### <span id="lifecycle-handler"></span> LifecycleHandler


> LifecycleHandler defines a specific action that should be taken in a lifecycle
hook. One and only one of the fields, except TCPSocket must be specified.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| exec | [ExecAction](#exec-action)| `ExecAction` |  | |  |  |
| httpGet | [HTTPGetAction](#http-get-action)| `HTTPGetAction` |  | |  |  |
| sleep | [SleepAction](#sleep-action)| `SleepAction` |  | |  |  |
| tcpSocket | [TCPSocketAction](#tcp-socket-action)| `TCPSocketAction` |  | |  |  |



### <span id="lifecycle-hook"></span> LifecycleHook


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| arguments | [Arguments](#arguments)| `Arguments` |  | |  |  |
| expression | string| `string` |  | | Expression is a condition expression for when a node will be retried. If it evaluates to false, the node will not</br>be retried and the retry strategy will be ignored |  |
| template | string| `string` |  | | Template is the name of the template to execute by the hook |  |
| templateRef | [TemplateRef](#template-ref)| `TemplateRef` |  | |  |  |



### <span id="lifecycle-hooks"></span> LifecycleHooks


  

[LifecycleHooks](#lifecycle-hooks)

### <span id="local-object-reference"></span> LocalObjectReference


> New uses of this type are discouraged because of difficulty describing its usage when embedded in APIs.
1. Invalid usage help.  It is impossible to add specific help for individual usage.  In most embedded usages, there are particular
restrictions like, "must refer only to types A and B" or "UID not honored" or "name must be restricted".
Those cannot be well described when embedded.
2. Inconsistent validation.  Because the usages are different, the validation rules are different by usage, which makes it hard for users to predict what will happen.
3. We cannot easily change it.  Because this type is embedded in many locations, updates to this type
will affect numerous schemas.  Don't make new APIs embed an underspecified API type they do not control.

Instead of using this type, create a locally provided and used type that is well-focused on your reference.
For example, ServiceReferences for admission registration: https://github.com/kubernetes/api/blob/release-1.17/admissionregistration/v1/types.go#L533 .
+structType=atomic
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | | Name of the referent.</br>This field is effectively required, but due to backwards compatibility is</br>allowed to be empty. Instances of this type with an empty value here are</br>almost certainly wrong.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</br>+optional</br>+default=""</br>+kubebuilder:default=""</br>TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |  |



### <span id="managed-fields-entry"></span> ManagedFieldsEntry


> ManagedFieldsEntry is a workflow-id, a FieldSet and the group version of the resource
that the fieldset applies to.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| apiVersion | string| `string` |  | | APIVersion defines the version of this resource that this field set</br>applies to. The format is "group/version" just like the top-level</br>APIVersion field. It is necessary to track the version of a field</br>set because it cannot be automatically converted. |  |
| fieldsType | string| `string` |  | | FieldsType is the discriminator for the different fields format and version.</br>There is currently only one possible value: "FieldsV1" |  |
| fieldsV1 | [FieldsV1](#fields-v1)| `FieldsV1` |  | |  |  |
| manager | string| `string` |  | | Manager is an identifier of the workflow managing these fields. |  |
| operation | [ManagedFieldsOperationType](#managed-fields-operation-type)| `ManagedFieldsOperationType` |  | |  |  |
| subresource | string| `string` |  | | Subresource is the name of the subresource used to update that object, or</br>empty string if the object was updated through the main resource. The</br>value of this field is used to distinguish between managers, even if they</br>share the same name. For example, a status update will be distinct from a</br>regular update using the same manager name.</br>Note that the APIVersion field is not related to the Subresource field and</br>it always corresponds to the version of the main resource. |  |
| time | string| `string` |  | | Time is the timestamp of when the ManagedFields entry was added. The</br>timestamp will also be updated if a field is added, the manager</br>changes any of the owned fields value or removes a field. The</br>timestamp does not update when a field is removed from the entry</br>because another manager took it over.</br>+optional |  |



### <span id="managed-fields-operation-type"></span> ManagedFieldsOperationType


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| ManagedFieldsOperationType | string| string | |  |  |



### <span id="manifest-from"></span> ManifestFrom


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| artifact | [Artifact](#artifact)| `Artifact` |  | |  |  |



### <span id="memoize"></span> Memoize


> Memoization enables caching for the Outputs of the template
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| cache | [Cache](#cache)| `Cache` |  | |  |  |
| key | string| `string` |  | | Key is the key to use as the caching key |  |
| maxAge | string| `string` |  | | MaxAge is the maximum age (e.g. "180s", "24h") of an entry that is still considered valid. If an entry is older</br>than the MaxAge, it will be ignored. |  |



### <span id="metadata"></span> Metadata


> Pod metdata
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| annotations | map of string| `map[string]string` |  | |  |  |
| labels | map of string| `map[string]string` |  | |  |  |



### <span id="metric-label"></span> MetricLabel


> MetricLabel is a single label for a prometheus metric
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| key | string| `string` |  | |  |  |
| value | string| `string` |  | |  |  |



### <span id="metrics"></span> Metrics


> Metrics are a list of metrics emitted from a Workflow/Template
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| prometheus | [][Prometheus](#prometheus)| `[]*Prometheus` |  | | Prometheus is a list of prometheus metrics to be emitted |  |



### <span id="mount-propagation-mode"></span> MountPropagationMode


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| MountPropagationMode | string| string | | +enum |  |



### <span id="mutex"></span> Mutex


> Mutex holds Mutex configuration
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | | name of the mutex |  |
| namespace | string| `string` |  | `"[namespace of workflow]"`|  |  |



### <span id="n-f-s-volume-source"></span> NFSVolumeSource


> NFS volumes do not support ownership management or SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| path | string| `string` |  | | path that is exported by the NFS server.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs |  |
| readOnly | boolean| `bool` |  | | readOnly here will force the NFS export to be mounted with read-only permissions.</br>Defaults to false.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs</br>+optional |  |
| server | string| `string` |  | | server is the hostname or IP address of the NFS server.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs |  |



### <span id="node-affinity"></span> NodeAffinity


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| preferredDuringSchedulingIgnoredDuringExecution | [][PreferredSchedulingTerm](#preferred-scheduling-term)| `[]*PreferredSchedulingTerm` |  | | The scheduler will prefer to schedule pods to nodes that satisfy</br>the affinity expressions specified by this field, but it may choose</br>a node that violates one or more of the expressions. The node that is</br>most preferred is the one with the greatest sum of weights, i.e.</br>for each node that meets all of the scheduling requirements (resource</br>request, requiredDuringScheduling affinity expressions, etc.),</br>compute a sum by iterating through the elements of this field and adding</br>"weight" to the sum if the node matches the corresponding matchExpressions; the</br>node(s) with the highest sum are the most preferred.</br>+optional</br>+listType=atomic |  |
| requiredDuringSchedulingIgnoredDuringExecution | [NodeSelector](#node-selector)| `NodeSelector` |  | |  |  |



### <span id="node-phase"></span> NodePhase


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| NodePhase | string| string | |  |  |



### <span id="node-result"></span> NodeResult


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| message | string| `string` |  | |  |  |
| outputs | [Outputs](#outputs)| `Outputs` |  | |  |  |
| phase | [NodePhase](#node-phase)| `NodePhase` |  | |  |  |
| progress | [Progress](#progress)| `Progress` |  | |  |  |



### <span id="node-selector"></span> NodeSelector


> A node selector represents the union of the results of one or more label queries
over a set of nodes; that is, it represents the OR of the selectors represented
by the node selector terms.
+structType=atomic
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| nodeSelectorTerms | [][NodeSelectorTerm](#node-selector-term)| `[]*NodeSelectorTerm` |  | | Required. A list of node selector terms. The terms are ORed.</br>+listType=atomic |  |



### <span id="node-selector-operator"></span> NodeSelectorOperator


> A node selector operator is the set of operators that can be used in
a node selector requirement.
+enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| NodeSelectorOperator | string| string | | A node selector operator is the set of operators that can be used in</br>a node selector requirement.</br>+enum |  |



### <span id="node-selector-requirement"></span> NodeSelectorRequirement


> A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| key | string| `string` |  | | The label key that the selector applies to. |  |
| operator | [NodeSelectorOperator](#node-selector-operator)| `NodeSelectorOperator` |  | |  |  |
| values | []string| `[]string` |  | | An array of string values. If the operator is In or NotIn,</br>the values array must be non-empty. If the operator is Exists or DoesNotExist,</br>the values array must be empty. If the operator is Gt or Lt, the values</br>array must have a single element, which will be interpreted as an integer.</br>This array is replaced during a strategic merge patch.</br>+optional</br>+listType=atomic |  |



### <span id="node-selector-term"></span> NodeSelectorTerm


> A null or empty node selector term matches no objects. The requirements of
them are ANDed.
The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.
+structType=atomic
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| matchExpressions | [][NodeSelectorRequirement](#node-selector-requirement)| `[]*NodeSelectorRequirement` |  | | A list of node selector requirements by node's labels.</br>+optional</br>+listType=atomic |  |
| matchFields | [][NodeSelectorRequirement](#node-selector-requirement)| `[]*NodeSelectorRequirement` |  | | A list of node selector requirements by node's fields.</br>+optional</br>+listType=atomic |  |



### <span id="none-strategy"></span> NoneStrategy


> NoneStrategy indicates to skip tar process and upload the files or directory tree as independent
files. Note that if the artifact is a directory, the artifact driver must support the ability to
save/load the directory appropriately.
  



[interface{}](#interface)

### <span id="o-auth2-auth"></span> OAuth2Auth


> OAuth2Auth holds all information for client authentication via OAuth2 tokens
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| clientIDSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| clientSecretSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| endpointParams | [][OAuth2EndpointParam](#o-auth2-endpoint-param)| `[]*OAuth2EndpointParam` |  | |  |  |
| scopes | []string| `[]string` |  | |  |  |
| tokenURLSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |



### <span id="o-auth2-endpoint-param"></span> OAuth2EndpointParam


> EndpointParam is for requesting optional fields that should be sent in the oauth request
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| key | string| `string` |  | | Name is the header name |  |
| value | string| `string` |  | | Value is the literal value to use for the header |  |



### <span id="o-s-s-artifact"></span> OSSArtifact


> OSSArtifact is the location of an Alibaba Cloud OSS artifact
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| accessKeySecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| bucket | string| `string` |  | | Bucket is the name of the bucket |  |
| createBucketIfNotPresent | boolean| `bool` |  | | CreateBucketIfNotPresent tells the driver to attempt to create the OSS bucket for output artifacts, if it doesn't exist |  |
| endpoint | string| `string` |  | | Endpoint is the hostname of the bucket endpoint |  |
| key | string| `string` |  | | Key is the path in the bucket where the artifact resides |  |
| lifecycleRule | [OSSLifecycleRule](#o-s-s-lifecycle-rule)| `OSSLifecycleRule` |  | |  |  |
| secretKeySecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| securityToken | string| `string` |  | | SecurityToken is the user's temporary security token. For more details, check out: https://www.alibabacloud.com/help/doc-detail/100624.htm |  |
| useSDKCreds | boolean| `bool` |  | | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. |  |



### <span id="o-s-s-lifecycle-rule"></span> OSSLifecycleRule


> OSSLifecycleRule specifies how to manage bucket's lifecycle
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| markDeletionAfterDays | int32 (formatted integer)| `int32` |  | | MarkDeletionAfterDays is the number of days before we delete objects in the bucket |  |
| markInfrequentAccessAfterDays | int32 (formatted integer)| `int32` |  | | MarkInfrequentAccessAfterDays is the number of days before we convert the objects in the bucket to Infrequent Access (IA) storage type |  |



### <span id="object-field-selector"></span> ObjectFieldSelector


> +structType=atomic
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| apiVersion | string| `string` |  | | Version of the schema the FieldPath is written in terms of, defaults to "v1".</br>+optional |  |
| fieldPath | string| `string` |  | | Path of the field to select in the specified API version. |  |



### <span id="object-meta"></span> ObjectMeta


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | |  |  |
| namespace | string| `string` |  | |  |  |
| uid | string| `string` |  | |  |  |



### <span id="outputs"></span> Outputs


> Outputs hold parameters, artifacts, and results from a step
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| artifacts | [Artifacts](#artifacts)| `Artifacts` |  | |  |  |
| exitCode | string| `string` |  | | ExitCode holds the exit code of a script template |  |
| parameters | [][Parameter](#parameter)| `[]*Parameter` |  | | Parameters holds the list of output parameters produced by a step</br>+patchStrategy=merge</br>+patchMergeKey=name |  |
| result | string| `string` |  | | Result holds the result (stdout) of a script template |  |



### <span id="owner-reference"></span> OwnerReference


> OwnerReference contains enough information to let you identify an owning
object. An owning object must be in the same namespace as the dependent, or
be cluster-scoped, so there is no namespace field.
+structType=atomic
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| apiVersion | string| `string` |  | | API version of the referent. |  |
| blockOwnerDeletion | boolean| `bool` |  | | If true, AND if the owner has the "foregroundDeletion" finalizer, then</br>the owner cannot be deleted from the key-value store until this</br>reference is removed.</br>See https://kubernetes.io/docs/concepts/architecture/garbage-collection/#foreground-deletion</br>for how the garbage collector interacts with this field and enforces the foreground deletion.</br>Defaults to false.</br>To set this field, a user needs "delete" permission of the owner,</br>otherwise 422 (Unprocessable Entity) will be returned.</br>+optional |  |
| controller | boolean| `bool` |  | | If true, this reference points to the managing controller.</br>+optional |  |
| kind | string| `string` |  | | Kind of the referent.</br>More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |  |
| name | string| `string` |  | | Name of the referent.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names |  |
| uid | [UID](#uid)| `UID` |  | |  |  |



### <span id="parallel-steps"></span> ParallelSteps


> +kubebuilder:validation:Type=array
  



[interface{}](#interface)

### <span id="parameter"></span> Parameter


> Parameter indicate a passed string parameter to a service template with an optional default value
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| default | [AnyString](#any-string)| `AnyString` |  | |  |  |
| description | [AnyString](#any-string)| `AnyString` |  | |  |  |
| enum | [][AnyString](#any-string)| `[]AnyString` |  | | Enum holds a list of string values to choose from, for the actual value of the parameter |  |
| globalName | string| `string` |  | | GlobalName exports an output parameter to the global scope, making it available as</br>'{{workflow.outputs.parameters.XXXX}} and in workflow.status.outputs.parameters |  |
| name | string| `string` |  | | Name is the parameter name |  |
| value | [AnyString](#any-string)| `AnyString` |  | |  |  |
| valueFrom | [ValueFrom](#value-from)| `ValueFrom` |  | |  |  |



### <span id="persistent-volume-access-mode"></span> PersistentVolumeAccessMode


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| PersistentVolumeAccessMode | string| string | | +enum |  |



### <span id="persistent-volume-claim-spec"></span> PersistentVolumeClaimSpec


> PersistentVolumeClaimSpec describes the common attributes of storage devices
and allows a Source for provider-specific attributes
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| accessModes | [][PersistentVolumeAccessMode](#persistent-volume-access-mode)| `[]PersistentVolumeAccessMode` |  | | accessModes contains the desired access modes the volume should have.</br>More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1</br>+optional</br>+listType=atomic |  |
| dataSource | [TypedLocalObjectReference](#typed-local-object-reference)| `TypedLocalObjectReference` |  | |  |  |
| dataSourceRef | [TypedObjectReference](#typed-object-reference)| `TypedObjectReference` |  | |  |  |
| resources | [VolumeResourceRequirements](#volume-resource-requirements)| `VolumeResourceRequirements` |  | |  |  |
| selector | [LabelSelector](#label-selector)| `LabelSelector` |  | |  |  |
| storageClassName | string| `string` |  | | storageClassName is the name of the StorageClass required by the claim.</br>More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1</br>+optional |  |
| volumeAttributesClassName | string| `string` |  | | volumeAttributesClassName may be used to set the VolumeAttributesClass used by this claim.</br>If specified, the CSI driver will create or update the volume with the attributes defined</br>in the corresponding VolumeAttributesClass. This has a different purpose than storageClassName,</br>it can be changed after the claim is created. An empty string value means that no VolumeAttributesClass</br>will be applied to the claim but it's not allowed to reset this field to empty string once it is set.</br>If unspecified and the PersistentVolumeClaim is unbound, the default VolumeAttributesClass</br>will be set by the persistentvolume controller if it exists.</br>If the resource referred to by volumeAttributesClass does not exist, this PersistentVolumeClaim will be</br>set to a Pending state, as reflected by the modifyVolumeStatus field, until such as a resource</br>exists.</br>More info: https://kubernetes.io/docs/concepts/storage/volume-attributes-classes/</br>(Beta) Using this field requires the VolumeAttributesClass feature gate to be enabled (off by default).</br>+featureGate=VolumeAttributesClass</br>+optional |  |
| volumeMode | [PersistentVolumeMode](#persistent-volume-mode)| `PersistentVolumeMode` |  | |  |  |
| volumeName | string| `string` |  | | volumeName is the binding reference to the PersistentVolume backing this claim.</br>+optional |  |



### <span id="persistent-volume-claim-template"></span> PersistentVolumeClaimTemplate


> PersistentVolumeClaimTemplate is used to produce
PersistentVolumeClaim objects as part of an EphemeralVolumeSource.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| annotations | map of string| `map[string]string` |  | | Annotations is an unstructured key value map stored with a resource that may be</br>set by external tools to store and retrieve arbitrary metadata. They are not</br>queryable and should be preserved when modifying objects.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations</br>+optional |  |
| creationTimestamp | string| `string` |  | | CreationTimestamp is a timestamp representing the server time when this object was</br>created. It is not guaranteed to be set in happens-before order across separate operations.</br>Clients may not set this value. It is represented in RFC3339 form and is in UTC.</br></br>Populated by the system.</br>Read-only.</br>Null for lists.</br>More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</br>+optional |  |
| deletionGracePeriodSeconds | int64 (formatted integer)| `int64` |  | | Number of seconds allowed for this object to gracefully terminate before</br>it will be removed from the system. Only set when deletionTimestamp is also set.</br>May only be shortened.</br>Read-only.</br>+optional |  |
| deletionTimestamp | string| `string` |  | | DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This</br>field is set by the server when a graceful deletion is requested by the user, and is not</br>directly settable by a client. The resource is expected to be deleted (no longer visible</br>from resource lists, and not reachable by name) after the time in this field, once the</br>finalizers list is empty. As long as the finalizers list contains items, deletion is blocked.</br>Once the deletionTimestamp is set, this value may not be unset or be set further into the</br>future, although it may be shortened or the resource may be deleted prior to this time.</br>For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react</br>by sending a graceful termination signal to the containers in the pod. After that 30 seconds,</br>the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup,</br>remove the pod from the API. In the presence of network partitions, this object may still</br>exist after this timestamp, until an administrator or automated process can determine the</br>resource is fully terminated.</br>If not set, graceful deletion of the object has not been requested.</br></br>Populated by the system when a graceful deletion is requested.</br>Read-only.</br>More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata</br>+optional |  |
| finalizers | []string| `[]string` |  | | Must be empty before the object is deleted from the registry. Each entry</br>is an identifier for the responsible component that will remove the entry</br>from the list. If the deletionTimestamp of the object is non-nil, entries</br>in this list can only be removed.</br>Finalizers may be processed and removed in any order.  Order is NOT enforced</br>because it introduces significant risk of stuck finalizers.</br>finalizers is a shared field, any actor with permission can reorder it.</br>If the finalizer list is processed in order, then this can lead to a situation</br>in which the component responsible for the first finalizer in the list is</br>waiting for a signal (field value, external system, or other) produced by a</br>component responsible for a finalizer later in the list, resulting in a deadlock.</br>Without enforced ordering finalizers are free to order amongst themselves and</br>are not vulnerable to ordering changes in the list.</br>+optional</br>+patchStrategy=merge</br>+listType=set |  |
| generateName | string| `string` |  | | GenerateName is an optional prefix, used by the server, to generate a unique</br>name ONLY IF the Name field has not been provided.</br>If this field is used, the name returned to the client will be different</br>than the name passed. This value will also be combined with a unique suffix.</br>The provided value has the same validation rules as the Name field,</br>and may be truncated by the length of the suffix required to make the value</br>unique on the server.</br></br>If this field is specified and the generated name exists, the server will return a 409.</br></br>Applied only if Name is not specified.</br>More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency</br>+optional |  |
| generation | int64 (formatted integer)| `int64` |  | | A sequence number representing a specific generation of the desired state.</br>Populated by the system. Read-only.</br>+optional |  |
| labels | map of string| `map[string]string` |  | | Map of string keys and values that can be used to organize and categorize</br>(scope and select) objects. May match selectors of replication controllers</br>and services.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels</br>+optional |  |
| managedFields | [][ManagedFieldsEntry](#managed-fields-entry)| `[]*ManagedFieldsEntry` |  | | ManagedFields maps workflow-id and version to the set of fields</br>that are managed by that workflow. This is mostly for internal</br>housekeeping, and users typically shouldn't need to set or</br>understand this field. A workflow can be the user's name, a</br>controller's name, or the name of a specific apply path like</br>"ci-cd". The set of fields is always in the version that the</br>workflow used when modifying the object.</br></br>+optional</br>+listType=atomic |  |
| name | string| `string` |  | | Name must be unique within a namespace. Is required when creating resources, although</br>some resources may allow a client to request the generation of an appropriate name</br>automatically. Name is primarily intended for creation idempotence and configuration</br>definition.</br>Cannot be updated.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names</br>+optional |  |
| namespace | string| `string` |  | | Namespace defines the space within which each name must be unique. An empty namespace is</br>equivalent to the "default" namespace, but "default" is the canonical representation.</br>Not all objects are required to be scoped to a namespace - the value of this field for</br>those objects will be empty.</br></br>Must be a DNS_LABEL.</br>Cannot be updated.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces</br>+optional |  |
| ownerReferences | [][OwnerReference](#owner-reference)| `[]*OwnerReference` |  | | List of objects depended by this object. If ALL objects in the list have</br>been deleted, this object will be garbage collected. If this object is managed by a controller,</br>then an entry in this list will point to this controller, with the controller field set to true.</br>There cannot be more than one managing controller.</br>+optional</br>+patchMergeKey=uid</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=uid |  |
| resourceVersion | string| `string` |  | | An opaque value that represents the internal version of this object that can</br>be used by clients to determine when objects have changed. May be used for optimistic</br>concurrency, change detection, and the watch operation on a resource or set of resources.</br>Clients must treat these values as opaque and passed unmodified back to the server.</br>They may only be valid for a particular resource or set of resources.</br></br>Populated by the system.</br>Read-only.</br>Value must be treated as opaque by clients and .</br>More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency</br>+optional |  |
| selfLink | string| `string` |  | | Deprecated: selfLink is a legacy read-only field that is no longer populated by the system.</br>+optional |  |
| spec | [PersistentVolumeClaimSpec](#persistent-volume-claim-spec)| `PersistentVolumeClaimSpec` |  | |  |  |
| uid | [UID](#uid)| `UID` |  | |  |  |



### <span id="persistent-volume-claim-volume-source"></span> PersistentVolumeClaimVolumeSource


> This volume finds the bound PV and mounts that volume for the pod. A
PersistentVolumeClaimVolumeSource is, essentially, a wrapper around another
type of volume that is owned by someone else (the system).
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| claimName | string| `string` |  | | claimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume.</br>More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims |  |
| readOnly | boolean| `bool` |  | | readOnly Will force the ReadOnly setting in VolumeMounts.</br>Default false.</br>+optional |  |



### <span id="persistent-volume-mode"></span> PersistentVolumeMode


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| PersistentVolumeMode | string| string | | +enum |  |



### <span id="photon-persistent-disk-volume-source"></span> PhotonPersistentDiskVolumeSource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fsType | string| `string` |  | | fsType is the filesystem type to mount.</br>Must be a filesystem type supported by the host operating system.</br>Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. |  |
| pdID | string| `string` |  | | pdID is the ID that identifies Photon Controller persistent disk |  |



### <span id="plugin"></span> Plugin


> Plugin is an Object with exactly one key
  



[interface{}](#interface)

### <span id="pod-affinity"></span> PodAffinity


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| preferredDuringSchedulingIgnoredDuringExecution | [][WeightedPodAffinityTerm](#weighted-pod-affinity-term)| `[]*WeightedPodAffinityTerm` |  | | The scheduler will prefer to schedule pods to nodes that satisfy</br>the affinity expressions specified by this field, but it may choose</br>a node that violates one or more of the expressions. The node that is</br>most preferred is the one with the greatest sum of weights, i.e.</br>for each node that meets all of the scheduling requirements (resource</br>request, requiredDuringScheduling affinity expressions, etc.),</br>compute a sum by iterating through the elements of this field and adding</br>"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the</br>node(s) with the highest sum are the most preferred.</br>+optional</br>+listType=atomic |  |
| requiredDuringSchedulingIgnoredDuringExecution | [][PodAffinityTerm](#pod-affinity-term)| `[]*PodAffinityTerm` |  | | If the affinity requirements specified by this field are not met at</br>scheduling time, the pod will not be scheduled onto the node.</br>If the affinity requirements specified by this field cease to be met</br>at some point during pod execution (e.g. due to a pod label update), the</br>system may or may not try to eventually evict the pod from its node.</br>When there are multiple elements, the lists of nodes corresponding to each</br>podAffinityTerm are intersected, i.e. all terms must be satisfied.</br>+optional</br>+listType=atomic |  |



### <span id="pod-affinity-term"></span> PodAffinityTerm


> Defines a set of pods (namely those matching the labelSelector
relative to the given namespace(s)) that this pod should be
co-located (affinity) or not co-located (anti-affinity) with,
where co-located is defined as running on a node whose value of
the label with key <topologyKey> matches that of any node on which
a pod of the set of pods is running
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| labelSelector | [LabelSelector](#label-selector)| `LabelSelector` |  | |  |  |
| matchLabelKeys | []string| `[]string` |  | | MatchLabelKeys is a set of pod label keys to select which pods will</br>be taken into consideration. The keys are used to lookup values from the</br>incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`</br>to select the group of existing pods which pods will be taken into consideration</br>for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming</br>pod labels will be ignored. The default value is empty.</br>The same key is forbidden to exist in both matchLabelKeys and labelSelector.</br>Also, matchLabelKeys cannot be set when labelSelector isn't set.</br>This is a beta field and requires enabling MatchLabelKeysInPodAffinity feature gate (enabled by default).</br></br>+listType=atomic</br>+optional |  |
| mismatchLabelKeys | []string| `[]string` |  | | MismatchLabelKeys is a set of pod label keys to select which pods will</br>be taken into consideration. The keys are used to lookup values from the</br>incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`</br>to select the group of existing pods which pods will be taken into consideration</br>for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming</br>pod labels will be ignored. The default value is empty.</br>The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.</br>Also, mismatchLabelKeys cannot be set when labelSelector isn't set.</br>This is a beta field and requires enabling MatchLabelKeysInPodAffinity feature gate (enabled by default).</br></br>+listType=atomic</br>+optional |  |
| namespaceSelector | [LabelSelector](#label-selector)| `LabelSelector` |  | |  |  |
| namespaces | []string| `[]string` |  | | namespaces specifies a static list of namespace names that the term applies to.</br>The term is applied to the union of the namespaces listed in this field</br>and the ones selected by namespaceSelector.</br>null or empty namespaces list and null namespaceSelector means "this pod's namespace".</br>+optional</br>+listType=atomic |  |
| topologyKey | string| `string` |  | | This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching</br>the labelSelector in the specified namespaces, where co-located is defined as running on a node</br>whose value of the label with key topologyKey matches that of any node on which any of the</br>selected pods is running.</br>Empty topologyKey is not allowed. |  |



### <span id="pod-anti-affinity"></span> PodAntiAffinity


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| preferredDuringSchedulingIgnoredDuringExecution | [][WeightedPodAffinityTerm](#weighted-pod-affinity-term)| `[]*WeightedPodAffinityTerm` |  | | The scheduler will prefer to schedule pods to nodes that satisfy</br>the anti-affinity expressions specified by this field, but it may choose</br>a node that violates one or more of the expressions. The node that is</br>most preferred is the one with the greatest sum of weights, i.e.</br>for each node that meets all of the scheduling requirements (resource</br>request, requiredDuringScheduling anti-affinity expressions, etc.),</br>compute a sum by iterating through the elements of this field and adding</br>"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the</br>node(s) with the highest sum are the most preferred.</br>+optional</br>+listType=atomic |  |
| requiredDuringSchedulingIgnoredDuringExecution | [][PodAffinityTerm](#pod-affinity-term)| `[]*PodAffinityTerm` |  | | If the anti-affinity requirements specified by this field are not met at</br>scheduling time, the pod will not be scheduled onto the node.</br>If the anti-affinity requirements specified by this field cease to be met</br>at some point during pod execution (e.g. due to a pod label update), the</br>system may or may not try to eventually evict the pod from its node.</br>When there are multiple elements, the lists of nodes corresponding to each</br>podAffinityTerm are intersected, i.e. all terms must be satisfied.</br>+optional</br>+listType=atomic |  |



### <span id="pod-f-s-group-change-policy"></span> PodFSGroupChangePolicy


> PodFSGroupChangePolicy holds policies that will be used for applying fsGroup to a volume
when volume is mounted.
+enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| PodFSGroupChangePolicy | string| string | | PodFSGroupChangePolicy holds policies that will be used for applying fsGroup to a volume</br>when volume is mounted.</br>+enum |  |



### <span id="pod-s-e-linux-change-policy"></span> PodSELinuxChangePolicy


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| PodSELinuxChangePolicy | string| string | |  |  |



### <span id="pod-security-context"></span> PodSecurityContext


> Some fields are also present in container.securityContext.  Field values of
container.securityContext take precedence over field values of PodSecurityContext.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| appArmorProfile | [AppArmorProfile](#app-armor-profile)| `AppArmorProfile` |  | |  |  |
| fsGroup | int64 (formatted integer)| `int64` |  | | A special supplemental group that applies to all containers in a pod.</br>Some volume types allow the Kubelet to change the ownership of that volume</br>to be owned by the pod:</br></br>1. The owning GID will be the FSGroup</br>2. The setgid bit is set (new files created in the volume will be owned by FSGroup)</br>3. The permission bits are OR'd with rw-rw----</br></br>If unset, the Kubelet will not modify the ownership and permissions of any volume.</br>Note that this field cannot be set when spec.os.name is windows.</br>+optional |  |
| fsGroupChangePolicy | [PodFSGroupChangePolicy](#pod-f-s-group-change-policy)| `PodFSGroupChangePolicy` |  | |  |  |
| runAsGroup | int64 (formatted integer)| `int64` |  | | The GID to run the entrypoint of the container process.</br>Uses runtime default if unset.</br>May also be set in SecurityContext.  If set in both SecurityContext and</br>PodSecurityContext, the value specified in SecurityContext takes precedence</br>for that container.</br>Note that this field cannot be set when spec.os.name is windows.</br>+optional |  |
| runAsNonRoot | boolean| `bool` |  | | Indicates that the container must run as a non-root user.</br>If true, the Kubelet will validate the image at runtime to ensure that it</br>does not run as UID 0 (root) and fail to start the container if it does.</br>If unset or false, no such validation will be performed.</br>May also be set in SecurityContext.  If set in both SecurityContext and</br>PodSecurityContext, the value specified in SecurityContext takes precedence.</br>+optional |  |
| runAsUser | int64 (formatted integer)| `int64` |  | | The UID to run the entrypoint of the container process.</br>Defaults to user specified in image metadata if unspecified.</br>May also be set in SecurityContext.  If set in both SecurityContext and</br>PodSecurityContext, the value specified in SecurityContext takes precedence</br>for that container.</br>Note that this field cannot be set when spec.os.name is windows.</br>+optional |  |
| seLinuxChangePolicy | [PodSELinuxChangePolicy](#pod-s-e-linux-change-policy)| `PodSELinuxChangePolicy` |  | |  |  |
| seLinuxOptions | [SELinuxOptions](#s-e-linux-options)| `SELinuxOptions` |  | |  |  |
| seccompProfile | [SeccompProfile](#seccomp-profile)| `SeccompProfile` |  | |  |  |
| supplementalGroups | []int64 (formatted integer)| `[]int64` |  | | A list of groups applied to the first process run in each container, in</br>addition to the container's primary GID and fsGroup (if specified).  If</br>the SupplementalGroupsPolicy feature is enabled, the</br>supplementalGroupsPolicy field determines whether these are in addition</br>to or instead of any group memberships defined in the container image.</br>If unspecified, no additional groups are added, though group memberships</br>defined in the container image may still be used, depending on the</br>supplementalGroupsPolicy field.</br>Note that this field cannot be set when spec.os.name is windows.</br>+optional</br>+listType=atomic |  |
| supplementalGroupsPolicy | [SupplementalGroupsPolicy](#supplemental-groups-policy)| `SupplementalGroupsPolicy` |  | |  |  |
| sysctls | [][Sysctl](#sysctl)| `[]*Sysctl` |  | | Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupported</br>sysctls (by the container runtime) might fail to launch.</br>Note that this field cannot be set when spec.os.name is windows.</br>+optional</br>+listType=atomic |  |
| windowsOptions | [WindowsSecurityContextOptions](#windows-security-context-options)| `WindowsSecurityContextOptions` |  | |  |  |



### <span id="portworx-volume-source"></span> PortworxVolumeSource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fsType | string| `string` |  | | fSType represents the filesystem type to mount</br>Must be a filesystem type supported by the host operating system.</br>Ex. "ext4", "xfs". Implicitly inferred to be "ext4" if unspecified. |  |
| readOnly | boolean| `bool` |  | | readOnly defaults to false (read/write). ReadOnly here will force</br>the ReadOnly setting in VolumeMounts.</br>+optional |  |
| volumeID | string| `string` |  | | volumeID uniquely identifies a Portworx volume |  |



### <span id="preferred-scheduling-term"></span> PreferredSchedulingTerm


> An empty preferred scheduling term matches all objects with implicit weight 0
(i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| preference | [NodeSelectorTerm](#node-selector-term)| `NodeSelectorTerm` |  | |  |  |
| weight | int32 (formatted integer)| `int32` |  | | Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100. |  |



### <span id="probe"></span> Probe


> Probe describes a health check to be performed against a container to determine whether it is
alive or ready to receive traffic.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| exec | [ExecAction](#exec-action)| `ExecAction` |  | |  |  |
| failureThreshold | int32 (formatted integer)| `int32` |  | | Minimum consecutive failures for the probe to be considered failed after having succeeded.</br>Defaults to 3. Minimum value is 1.</br>+optional |  |
| grpc | [GRPCAction](#g-rpc-action)| `GRPCAction` |  | |  |  |
| httpGet | [HTTPGetAction](#http-get-action)| `HTTPGetAction` |  | |  |  |
| initialDelaySeconds | int32 (formatted integer)| `int32` |  | | Number of seconds after the container has started before liveness probes are initiated.</br>More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes</br>+optional |  |
| periodSeconds | int32 (formatted integer)| `int32` |  | | How often (in seconds) to perform the probe.</br>Default to 10 seconds. Minimum value is 1.</br>+optional |  |
| successThreshold | int32 (formatted integer)| `int32` |  | | Minimum consecutive successes for the probe to be considered successful after having failed.</br>Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.</br>+optional |  |
| tcpSocket | [TCPSocketAction](#tcp-socket-action)| `TCPSocketAction` |  | |  |  |
| terminationGracePeriodSeconds | int64 (formatted integer)| `int64` |  | | Optional duration in seconds the pod needs to terminate gracefully upon probe failure.</br>The grace period is the duration in seconds after the processes running in the pod are sent</br>a termination signal and the time when the processes are forcibly halted with a kill signal.</br>Set this value longer than the expected cleanup time for your process.</br>If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this</br>value overrides the value provided by the pod spec.</br>Value must be non-negative integer. The value zero indicates stop immediately via</br>the kill signal (no opportunity to shut down).</br>This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate.</br>Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.</br>+optional |  |
| timeoutSeconds | int32 (formatted integer)| `int32` |  | | Number of seconds after which the probe times out.</br>Defaults to 1 second. Minimum value is 1.</br>More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes</br>+optional |  |



### <span id="proc-mount-type"></span> ProcMountType


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| ProcMountType | string| string | | +enum |  |



### <span id="progress"></span> Progress


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Progress | string| string | |  |  |



### <span id="projected-volume-source"></span> ProjectedVolumeSource


> Represents a projected volume source
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| defaultMode | int32 (formatted integer)| `int32` |  | | defaultMode are the mode bits used to set permissions on created files by default.</br>Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.</br>YAML accepts both octal and decimal values, JSON requires decimal values for mode bits.</br>Directories within the path are not affected by this setting.</br>This might be in conflict with other options that affect the file</br>mode, like fsGroup, and the result can be other mode bits set.</br>+optional |  |
| sources | [][VolumeProjection](#volume-projection)| `[]*VolumeProjection` |  | | sources is the list of volume projections. Each entry in this list</br>handles one source.</br>+optional</br>+listType=atomic |  |



### <span id="prometheus"></span> Prometheus


> Prometheus is a prometheus metric to be emitted
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| counter | [Counter](#counter)| `Counter` |  | |  |  |
| gauge | [Gauge](#gauge)| `Gauge` |  | |  |  |
| help | string| `string` |  | | Help is a string that describes the metric |  |
| histogram | [Histogram](#histogram)| `Histogram` |  | |  |  |
| labels | [][MetricLabel](#metric-label)| `[]*MetricLabel` |  | | Labels is a list of metric labels |  |
| name | string| `string` |  | | Name is the name of the metric |  |
| when | string| `string` |  | | When is a conditional statement that decides when to emit the metric |  |



### <span id="protocol"></span> Protocol


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Protocol | string| string | | +enum |  |



### <span id="pull-policy"></span> PullPolicy


> PullPolicy describes a policy for if/when to pull a container image
+enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| PullPolicy | string| string | | PullPolicy describes a policy for if/when to pull a container image</br>+enum |  |



### <span id="quantity"></span> Quantity


> The serialization format is:

```
<quantity>        ::= <signedNumber><suffix>

(Note that <suffix> may be empty, from the "" case in <decimalSI>.)

<digit>           ::= 0 | 1 | ... | 9
<digits>          ::= <digit> | <digit><digits>
<number>          ::= <digits> | <digits>.<digits> | <digits>. | .<digits>
<sign>            ::= "+" | "-"
<signedNumber>    ::= <number> | <sign><number>
<suffix>          ::= <binarySI> | <decimalExponent> | <decimalSI>
<binarySI>        ::= Ki | Mi | Gi | Ti | Pi | Ei

(International System of units; See: http://physics.nist.gov/cuu/Units/binary.html)

<decimalSI>       ::= m | "" | k | M | G | T | P | E

(Note that 1024 = 1Ki but 1000 = 1k; I didn't choose the capitalization.)

<decimalExponent> ::= "e" <signedNumber> | "E" <signedNumber>
```

No matter which of the three exponent forms is used, no quantity may represent
a number greater than 2^63-1 in magnitude, nor may it have more than 3 decimal
places. Numbers larger or more precise will be capped or rounded up.
(E.g.: 0.1m will rounded up to 1m.)
This may be extended in the future if we require larger or smaller quantities.

When a Quantity is parsed from a string, it will remember the type of suffix
it had, and will use the same type again when it is serialized.

Before serializing, Quantity will be put in "canonical form".
This means that Exponent/suffix will be adjusted up or down (with a
corresponding increase or decrease in Mantissa) such that:

No precision is lost
No fractional digits will be emitted
The exponent (or suffix) is as large as possible.

The sign will be omitted unless the number is negative.

Examples:

1.5 will be serialized as "1500m"
1.5Gi will be serialized as "1536Mi"

Note that the quantity will NEVER be internally represented by a
floating point number. That is the whole point of this exercise.

Non-canonical values will still parse as long as they are well formed,
but will be re-emitted in their canonical form. (So always use canonical
form, or don't diff.)

This format is intended to make it difficult to use these numbers without
writing some sort of special handling code in the hopes that that will
cause implementors to also use a fixed point implementation.

+protobuf=true
+protobuf.embed=string
+protobuf.options.marshal=false
+protobuf.options.(gogoproto.goproto_stringer)=false
+k8s:deepcopy-gen=true
+k8s:openapi-gen=true
  



[interface{}](#interface)

### <span id="quobyte-volume-source"></span> QuobyteVolumeSource


> Quobyte volumes do not support ownership management or SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| group | string| `string` |  | | group to map volume access to</br>Default is no group</br>+optional |  |
| readOnly | boolean| `bool` |  | | readOnly here will force the Quobyte volume to be mounted with read-only permissions.</br>Defaults to false.</br>+optional |  |
| registry | string| `string` |  | | registry represents a single or multiple Quobyte Registry services</br>specified as a string as host:port pair (multiple entries are separated with commas)</br>which acts as the central registry for volumes |  |
| tenant | string| `string` |  | | tenant owning the given Quobyte volume in the Backend</br>Used with dynamically provisioned Quobyte volumes, value is set by the plugin</br>+optional |  |
| user | string| `string` |  | | user to map volume access to</br>Defaults to serivceaccount user</br>+optional |  |
| volume | string| `string` |  | | volume is a string that references an already created Quobyte volume by name. |  |



### <span id="r-b-d-volume-source"></span> RBDVolumeSource


> RBD volumes support ownership management and SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fsType | string| `string` |  | | fsType is the filesystem type of the volume that you want to mount.</br>Tip: Ensure that the filesystem type is supported by the host operating system.</br>Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#rbd</br>TODO: how do we prevent errors in the filesystem from compromising the machine</br>+optional |  |
| image | string| `string` |  | | image is the rados image name.</br>More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it |  |
| keyring | string| `string` |  | | keyring is the path to key ring for RBDUser.</br>Default is /etc/ceph/keyring.</br>More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it</br>+optional</br>+default="/etc/ceph/keyring" |  |
| monitors | []string| `[]string` |  | | monitors is a collection of Ceph monitors.</br>More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it</br>+listType=atomic |  |
| pool | string| `string` |  | | pool is the rados pool name.</br>Default is rbd.</br>More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it</br>+optional</br>+default="rbd" |  |
| readOnly | boolean| `bool` |  | | readOnly here will force the ReadOnly setting in VolumeMounts.</br>Defaults to false.</br>More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it</br>+optional |  |
| secretRef | [LocalObjectReference](#local-object-reference)| `LocalObjectReference` |  | |  |  |
| user | string| `string` |  | | user is the rados user name.</br>Default is admin.</br>More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it</br>+optional</br>+default="admin" |  |



### <span id="raw-artifact"></span> RawArtifact


> RawArtifact allows raw string content to be placed as an artifact in a container
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| data | string| `string` |  | | Data is the string contents of the artifact |  |



### <span id="recursive-read-only-mode"></span> RecursiveReadOnlyMode


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| RecursiveReadOnlyMode | string| string | |  |  |



### <span id="resource-claim"></span> ResourceClaim


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | | Name must match the name of one entry in pod.spec.resourceClaims of</br>the Pod where this field is used. It makes that resource available</br>inside a container. |  |
| request | string| `string` |  | | Request is the name chosen for a request in the referenced claim.</br>If empty, everything from the claim is made available, otherwise</br>only the result of this request.</br></br>+optional |  |



### <span id="resource-field-selector"></span> ResourceFieldSelector


> ResourceFieldSelector represents container resources (cpu, memory) and their output format
+structType=atomic
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| containerName | string| `string` |  | | Container name: required for volumes, optional for env vars</br>+optional |  |
| divisor | [Quantity](#quantity)| `Quantity` |  | |  |  |
| resource | string| `string` |  | | Required: resource to select |  |



### <span id="resource-list"></span> ResourceList


  

[ResourceList](#resource-list)

### <span id="resource-name"></span> ResourceName


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| ResourceName | string| string | |  |  |



### <span id="resource-requirements"></span> ResourceRequirements


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| claims | [][ResourceClaim](#resource-claim)| `[]*ResourceClaim` |  | | Claims lists the names of resources, defined in spec.resourceClaims,</br>that are used by this container.</br></br>This is an alpha field and requires enabling the</br>DynamicResourceAllocation feature gate.</br></br>This field is immutable. It can only be set for containers.</br></br>+listType=map</br>+listMapKey=name</br>+featureGate=DynamicResourceAllocation</br>+optional |  |
| limits | [ResourceList](#resource-list)| `ResourceList` |  | |  |  |
| requests | [ResourceList](#resource-list)| `ResourceList` |  | |  |  |



### <span id="resource-resize-restart-policy"></span> ResourceResizeRestartPolicy


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| ResourceResizeRestartPolicy | string| string | |  |  |



### <span id="resource-template"></span> ResourceTemplate


> ResourceTemplate is a template subtype to manipulate kubernetes resources
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| action | string| `string` |  | | Action is the action to perform to the resource.</br>Must be one of: get, create, apply, delete, replace, patch |  |
| failureCondition | string| `string` |  | | FailureCondition is a label selector expression which describes the conditions</br>of the k8s resource in which the step was considered failed |  |
| flags | []string| `[]string` |  | | Flags is a set of additional options passed to kubectl before submitting a resource</br>I.e. to disable resource validation:</br>flags: [</br>"--validate=false"  # disable resource validation</br>] |  |
| manifest | string| `string` |  | | Manifest contains the kubernetes manifest |  |
| manifestFrom | [ManifestFrom](#manifest-from)| `ManifestFrom` |  | |  |  |
| mergeStrategy | string| `string` |  | | MergeStrategy is the strategy used to merge a patch. It defaults to "strategic"</br>Must be one of: strategic, merge, json |  |
| setOwnerReference | boolean| `bool` |  | | SetOwnerReference sets the reference to the workflow on the OwnerReference of generated resource. |  |
| successCondition | string| `string` |  | | SuccessCondition is a label selector expression which describes the conditions</br>of the k8s resource in which it is acceptable to proceed to the following step |  |



### <span id="retry-affinity"></span> RetryAffinity


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| nodeAntiAffinity | [RetryNodeAntiAffinity](#retry-node-anti-affinity)| `RetryNodeAntiAffinity` |  | |  |  |



### <span id="retry-node-anti-affinity"></span> RetryNodeAntiAffinity


> In order to prevent running steps on the same host, it uses "kubernetes.io/hostname".
  



[interface{}](#interface)

### <span id="retry-policy"></span> RetryPolicy


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| RetryPolicy | string| string | |  |  |



### <span id="retry-strategy"></span> RetryStrategy


> RetryStrategy provides controls on how to retry a workflow step
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| affinity | [RetryAffinity](#retry-affinity)| `RetryAffinity` |  | |  |  |
| backoff | [Backoff](#backoff)| `Backoff` |  | |  |  |
| expression | string| `string` |  | | Expression is a condition expression for when a node will be retried. If it evaluates to false, the node will not</br>be retried and the retry strategy will be ignored |  |
| limit | [IntOrString](#int-or-string)| `IntOrString` |  | |  |  |
| retryPolicy | [RetryPolicy](#retry-policy)| `RetryPolicy` |  | |  |  |



### <span id="s3-artifact"></span> S3Artifact


> S3Artifact is the location of an S3 artifact
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| accessKeySecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| bucket | string| `string` |  | | Bucket is the name of the bucket |  |
| caSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| createBucketIfNotPresent | [CreateS3BucketOptions](#create-s3-bucket-options)| `CreateS3BucketOptions` |  | |  |  |
| encryptionOptions | [S3EncryptionOptions](#s3-encryption-options)| `S3EncryptionOptions` |  | |  |  |
| endpoint | string| `string` |  | | Endpoint is the hostname of the bucket endpoint |  |
| insecure | boolean| `bool` |  | | Insecure will connect to the service with TLS |  |
| key | string| `string` |  | | Key is the key in the bucket where the artifact resides |  |
| region | string| `string` |  | | Region contains the optional bucket region |  |
| roleARN | string| `string` |  | | RoleARN is the Amazon Resource Name (ARN) of the role to assume. |  |
| secretKeySecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| sessionTokenSecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |
| useSDKCreds | boolean| `bool` |  | | UseSDKCreds tells the driver to figure out credentials based on sdk defaults. |  |



### <span id="s3-encryption-options"></span> S3EncryptionOptions


> S3EncryptionOptions used to determine encryption options during s3 operations
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| enableEncryption | boolean| `bool` |  | | EnableEncryption tells the driver to encrypt objects if set to true. If kmsKeyId and serverSideCustomerKeySecret are not set, SSE-S3 will be used |  |
| kmsEncryptionContext | string| `string` |  | | KmsEncryptionContext is a json blob that contains an encryption context. See https://docs.aws.amazon.com/kms/latest/developerguide/concepts.html#encrypt_context for more information |  |
| kmsKeyId | string| `string` |  | | KMSKeyId tells the driver to encrypt the object using the specified KMS Key. |  |
| serverSideCustomerKeySecret | [SecretKeySelector](#secret-key-selector)| `SecretKeySelector` |  | |  |  |



### <span id="s-e-linux-options"></span> SELinuxOptions


> SELinuxOptions are the labels to be applied to the container
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| level | string| `string` |  | | Level is SELinux level label that applies to the container.</br>+optional |  |
| role | string| `string` |  | | Role is a SELinux role label that applies to the container.</br>+optional |  |
| type | string| `string` |  | | Type is a SELinux type label that applies to the container.</br>+optional |  |
| user | string| `string` |  | | User is a SELinux user label that applies to the container.</br>+optional |  |



### <span id="scale-i-o-volume-source"></span> ScaleIOVolumeSource


> ScaleIOVolumeSource represents a persistent ScaleIO volume
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fsType | string| `string` |  | | fsType is the filesystem type to mount.</br>Must be a filesystem type supported by the host operating system.</br>Ex. "ext4", "xfs", "ntfs".</br>Default is "xfs".</br>+optional</br>+default="xfs" |  |
| gateway | string| `string` |  | | gateway is the host address of the ScaleIO API Gateway. |  |
| protectionDomain | string| `string` |  | | protectionDomain is the name of the ScaleIO Protection Domain for the configured storage.</br>+optional |  |
| readOnly | boolean| `bool` |  | | readOnly Defaults to false (read/write). ReadOnly here will force</br>the ReadOnly setting in VolumeMounts.</br>+optional |  |
| secretRef | [LocalObjectReference](#local-object-reference)| `LocalObjectReference` |  | |  |  |
| sslEnabled | boolean| `bool` |  | | sslEnabled Flag enable/disable SSL communication with Gateway, default false</br>+optional |  |
| storageMode | string| `string` |  | | storageMode indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned.</br>Default is ThinProvisioned.</br>+optional</br>+default="ThinProvisioned" |  |
| storagePool | string| `string` |  | | storagePool is the ScaleIO Storage Pool associated with the protection domain.</br>+optional |  |
| system | string| `string` |  | | system is the name of the storage system as configured in ScaleIO. |  |
| volumeName | string| `string` |  | | volumeName is the name of a volume already created in the ScaleIO system</br>that is associated with this volume source. |  |



### <span id="script-template"></span> ScriptTemplate


> ScriptTemplate is a template subtype to enable scripting through code steps
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| args | []string| `[]string` |  | | Arguments to the entrypoint.</br>The container image's CMD is used if this is not provided.</br>Variable references $(VAR_NAME) are expanded using the container's environment. If a variable</br>cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced</br>to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will</br>produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless</br>of whether the variable exists or not. Cannot be updated.</br>More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell</br>+optional</br>+listType=atomic |  |
| command | []string| `[]string` |  | | Entrypoint array. Not executed within a shell.</br>The container image's ENTRYPOINT is used if this is not provided.</br>Variable references $(VAR_NAME) are expanded using the container's environment. If a variable</br>cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced</br>to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will</br>produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless</br>of whether the variable exists or not. Cannot be updated.</br>More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell</br>+optional</br>+listType=atomic |  |
| env | [][EnvVar](#env-var)| `[]*EnvVar` |  | | List of environment variables to set in the container.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=name</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=name |  |
| envFrom | [][EnvFromSource](#env-from-source)| `[]*EnvFromSource` |  | | List of sources to populate environment variables in the container.</br>The keys defined within a source must be a C_IDENTIFIER. All invalid keys</br>will be reported as an event when the container is starting. When a key exists in multiple</br>sources, the value associated with the last source will take precedence.</br>Values defined by an Env with a duplicate key will take precedence.</br>Cannot be updated.</br>+optional</br>+listType=atomic |  |
| image | string| `string` |  | | Container image name.</br>More info: https://kubernetes.io/docs/concepts/containers/images</br>This field is optional to allow higher level config management to default or override</br>container images in workload controllers like Deployments and StatefulSets.</br>+optional |  |
| imagePullPolicy | [PullPolicy](#pull-policy)| `PullPolicy` |  | |  |  |
| lifecycle | [Lifecycle](#lifecycle)| `Lifecycle` |  | |  |  |
| livenessProbe | [Probe](#probe)| `Probe` |  | |  |  |
| name | string| `string` |  | | Name of the container specified as a DNS_LABEL.</br>Each container in a pod must have a unique name (DNS_LABEL).</br>Cannot be updated. |  |
| ports | [][ContainerPort](#container-port)| `[]*ContainerPort` |  | | List of ports to expose from the container. Not specifying a port here</br>DOES NOT prevent that port from being exposed. Any port which is</br>listening on the default "0.0.0.0" address inside a container will be</br>accessible from the network.</br>Modifying this array with strategic merge patch may corrupt the data.</br>For more information See https://github.com/kubernetes/kubernetes/issues/108255.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=containerPort</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=containerPort</br>+listMapKey=protocol |  |
| readinessProbe | [Probe](#probe)| `Probe` |  | |  |  |
| resizePolicy | [][ContainerResizePolicy](#container-resize-policy)| `[]*ContainerResizePolicy` |  | | Resources resize policy for the container.</br>+featureGate=InPlacePodVerticalScaling</br>+optional</br>+listType=atomic |  |
| resources | [ResourceRequirements](#resource-requirements)| `ResourceRequirements` |  | |  |  |
| restartPolicy | [ContainerRestartPolicy](#container-restart-policy)| `ContainerRestartPolicy` |  | |  |  |
| securityContext | [SecurityContext](#security-context)| `SecurityContext` |  | |  |  |
| source | string| `string` |  | | Source contains the source code of the script to execute |  |
| startupProbe | [Probe](#probe)| `Probe` |  | |  |  |
| stdin | boolean| `bool` |  | | Whether this container should allocate a buffer for stdin in the container runtime. If this</br>is not set, reads from stdin in the container will always result in EOF.</br>Default is false.</br>+optional |  |
| stdinOnce | boolean| `bool` |  | | Whether the container runtime should close the stdin channel after it has been opened by</br>a single attach. When stdin is true the stdin stream will remain open across multiple attach</br>sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the</br>first client attaches to stdin, and then remains open and accepts data until the client disconnects,</br>at which time stdin is closed and remains closed until the container is restarted. If this</br>flag is false, a container processes that reads from stdin will never receive an EOF.</br>Default is false</br>+optional |  |
| terminationMessagePath | string| `string` |  | | Optional: Path at which the file to which the container's termination message</br>will be written is mounted into the container's filesystem.</br>Message written is intended to be brief final status, such as an assertion failure message.</br>Will be truncated by the node if greater than 4096 bytes. The total message length across</br>all containers will be limited to 12kb.</br>Defaults to /dev/termination-log.</br>Cannot be updated.</br>+optional |  |
| terminationMessagePolicy | [TerminationMessagePolicy](#termination-message-policy)| `TerminationMessagePolicy` |  | |  |  |
| tty | boolean| `bool` |  | | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true.</br>Default is false.</br>+optional |  |
| volumeDevices | [][VolumeDevice](#volume-device)| `[]*VolumeDevice` |  | | volumeDevices is the list of block devices to be used by the container.</br>+patchMergeKey=devicePath</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=devicePath</br>+optional |  |
| volumeMounts | [][VolumeMount](#volume-mount)| `[]*VolumeMount` |  | | Pod volumes to mount into the container's filesystem.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=mountPath</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=mountPath |  |
| workingDir | string| `string` |  | | Container's working directory.</br>If not specified, the container runtime's default will be used, which</br>might be configured in the container image.</br>Cannot be updated.</br>+optional |  |



### <span id="seccomp-profile"></span> SeccompProfile


> Only one profile source may be set.
+union
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| localhostProfile | string| `string` |  | | localhostProfile indicates a profile defined in a file on the node should be used.</br>The profile must be preconfigured on the node to work.</br>Must be a descending path, relative to the kubelet's configured seccomp profile location.</br>Must be set if type is "Localhost". Must NOT be set for any other type.</br>+optional |  |
| type | [SeccompProfileType](#seccomp-profile-type)| `SeccompProfileType` |  | |  |  |



### <span id="seccomp-profile-type"></span> SeccompProfileType


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| SeccompProfileType | string| string | | +enum |  |



### <span id="secret-env-source"></span> SecretEnvSource


> The contents of the target Secret's Data field will represent the
key-value pairs as environment variables.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | | Name of the referent.</br>This field is effectively required, but due to backwards compatibility is</br>allowed to be empty. Instances of this type with an empty value here are</br>almost certainly wrong.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</br>+optional</br>+default=""</br>+kubebuilder:default=""</br>TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |  |
| optional | boolean| `bool` |  | | Specify whether the Secret must be defined</br>+optional |  |



### <span id="secret-key-selector"></span> SecretKeySelector


> +structType=atomic
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| key | string| `string` |  | | The key of the secret to select from.  Must be a valid secret key. |  |
| name | string| `string` |  | | Name of the referent.</br>This field is effectively required, but due to backwards compatibility is</br>allowed to be empty. Instances of this type with an empty value here are</br>almost certainly wrong.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</br>+optional</br>+default=""</br>+kubebuilder:default=""</br>TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |  |
| optional | boolean| `bool` |  | | Specify whether the Secret or its key must be defined</br>+optional |  |



### <span id="secret-projection"></span> SecretProjection


> The contents of the target Secret's Data field will be presented in a
projected volume as files using the keys in the Data field as the file names.
Note that this is identical to a secret volume source without the default
mode.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| items | [][KeyToPath](#key-to-path)| `[]*KeyToPath` |  | | items if unspecified, each key-value pair in the Data field of the referenced</br>Secret will be projected into the volume as a file whose name is the</br>key and content is the value. If specified, the listed keys will be</br>projected into the specified paths, and unlisted keys will not be</br>present. If a key is specified which is not present in the Secret,</br>the volume setup will error unless it is marked optional. Paths must be</br>relative and may not contain the '..' path or start with '..'.</br>+optional</br>+listType=atomic |  |
| name | string| `string` |  | | Name of the referent.</br>This field is effectively required, but due to backwards compatibility is</br>allowed to be empty. Instances of this type with an empty value here are</br>almost certainly wrong.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</br>+optional</br>+default=""</br>+kubebuilder:default=""</br>TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |  |
| optional | boolean| `bool` |  | | optional field specify whether the Secret or its key must be defined</br>+optional |  |



### <span id="secret-volume-source"></span> SecretVolumeSource


> The contents of the target Secret's Data field will be presented in a volume
as files using the keys in the Data field as the file names.
Secret volumes support ownership management and SELinux relabeling.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| defaultMode | int32 (formatted integer)| `int32` |  | | defaultMode is Optional: mode bits used to set permissions on created files by default.</br>Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.</br>YAML accepts both octal and decimal values, JSON requires decimal values</br>for mode bits. Defaults to 0644.</br>Directories within the path are not affected by this setting.</br>This might be in conflict with other options that affect the file</br>mode, like fsGroup, and the result can be other mode bits set.</br>+optional |  |
| items | [][KeyToPath](#key-to-path)| `[]*KeyToPath` |  | | items If unspecified, each key-value pair in the Data field of the referenced</br>Secret will be projected into the volume as a file whose name is the</br>key and content is the value. If specified, the listed keys will be</br>projected into the specified paths, and unlisted keys will not be</br>present. If a key is specified which is not present in the Secret,</br>the volume setup will error unless it is marked optional. Paths must be</br>relative and may not contain the '..' path or start with '..'.</br>+optional</br>+listType=atomic |  |
| optional | boolean| `bool` |  | | optional field specify whether the Secret or its keys must be defined</br>+optional |  |
| secretName | string| `string` |  | | secretName is the name of the secret in the pod's namespace to use.</br>More info: https://kubernetes.io/docs/concepts/storage/volumes#secret</br>+optional |  |



### <span id="security-context"></span> SecurityContext


> Some fields are present in both SecurityContext and PodSecurityContext.  When both
are set, the values in SecurityContext take precedence.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| allowPrivilegeEscalation | boolean| `bool` |  | | AllowPrivilegeEscalation controls whether a process can gain more</br>privileges than its parent process. This bool directly controls if</br>the no_new_privs flag will be set on the container process.</br>AllowPrivilegeEscalation is true always when the container is:</br>1) run as Privileged</br>2) has CAP_SYS_ADMIN</br>Note that this field cannot be set when spec.os.name is windows.</br>+optional |  |
| appArmorProfile | [AppArmorProfile](#app-armor-profile)| `AppArmorProfile` |  | |  |  |
| capabilities | [Capabilities](#capabilities)| `Capabilities` |  | |  |  |
| privileged | boolean| `bool` |  | | Run container in privileged mode.</br>Processes in privileged containers are essentially equivalent to root on the host.</br>Defaults to false.</br>Note that this field cannot be set when spec.os.name is windows.</br>+optional |  |
| procMount | [ProcMountType](#proc-mount-type)| `ProcMountType` |  | |  |  |
| readOnlyRootFilesystem | boolean| `bool` |  | | Whether this container has a read-only root filesystem.</br>Default is false.</br>Note that this field cannot be set when spec.os.name is windows.</br>+optional |  |
| runAsGroup | int64 (formatted integer)| `int64` |  | | The GID to run the entrypoint of the container process.</br>Uses runtime default if unset.</br>May also be set in PodSecurityContext.  If set in both SecurityContext and</br>PodSecurityContext, the value specified in SecurityContext takes precedence.</br>Note that this field cannot be set when spec.os.name is windows.</br>+optional |  |
| runAsNonRoot | boolean| `bool` |  | | Indicates that the container must run as a non-root user.</br>If true, the Kubelet will validate the image at runtime to ensure that it</br>does not run as UID 0 (root) and fail to start the container if it does.</br>If unset or false, no such validation will be performed.</br>May also be set in PodSecurityContext.  If set in both SecurityContext and</br>PodSecurityContext, the value specified in SecurityContext takes precedence.</br>+optional |  |
| runAsUser | int64 (formatted integer)| `int64` |  | | The UID to run the entrypoint of the container process.</br>Defaults to user specified in image metadata if unspecified.</br>May also be set in PodSecurityContext.  If set in both SecurityContext and</br>PodSecurityContext, the value specified in SecurityContext takes precedence.</br>Note that this field cannot be set when spec.os.name is windows.</br>+optional |  |
| seLinuxOptions | [SELinuxOptions](#s-e-linux-options)| `SELinuxOptions` |  | |  |  |
| seccompProfile | [SeccompProfile](#seccomp-profile)| `SeccompProfile` |  | |  |  |
| windowsOptions | [WindowsSecurityContextOptions](#windows-security-context-options)| `WindowsSecurityContextOptions` |  | |  |  |



### <span id="semaphore-ref"></span> SemaphoreRef


> SemaphoreRef is a reference of Semaphore
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| configMapKeyRef | [ConfigMapKeySelector](#config-map-key-selector)| `ConfigMapKeySelector` |  | |  |  |
| namespace | string| `string` |  | `"[namespace of workflow]"`|  |  |



### <span id="sequence"></span> Sequence


> Sequence expands a workflow step into numeric range
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| count | [IntOrString](#int-or-string)| `IntOrString` |  | |  |  |
| end | [IntOrString](#int-or-string)| `IntOrString` |  | |  |  |
| format | string| `string` |  | | Format is a printf format string to format the value in the sequence |  |
| start | [IntOrString](#int-or-string)| `IntOrString` |  | |  |  |



### <span id="service-account-token-projection"></span> ServiceAccountTokenProjection


> ServiceAccountTokenProjection represents a projected service account token
volume. This projection can be used to insert a service account token into
the pods runtime filesystem for use against APIs (Kubernetes API Server or
otherwise).
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| audience | string| `string` |  | | audience is the intended audience of the token. A recipient of a token</br>must identify itself with an identifier specified in the audience of the</br>token, and otherwise should reject the token. The audience defaults to the</br>identifier of the apiserver.</br>+optional |  |
| expirationSeconds | int64 (formatted integer)| `int64` |  | | expirationSeconds is the requested duration of validity of the service</br>account token. As the token approaches expiration, the kubelet volume</br>plugin will proactively rotate the service account token. The kubelet will</br>start trying to rotate the token if the token is older than 80 percent of</br>its time to live or if the token is older than 24 hours.Defaults to 1 hour</br>and must be at least 10 minutes.</br>+optional |  |
| path | string| `string` |  | | path is the path relative to the mount point of the file to project the</br>token into. |  |



### <span id="sleep-action"></span> SleepAction


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| seconds | int64 (formatted integer)| `int64` |  | | Seconds is the number of seconds to sleep. |  |



### <span id="storage-medium"></span> StorageMedium


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| StorageMedium | string| string | |  |  |



### <span id="storage-o-s-volume-source"></span> StorageOSVolumeSource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fsType | string| `string` |  | | fsType is the filesystem type to mount.</br>Must be a filesystem type supported by the host operating system.</br>Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.</br>+optional |  |
| readOnly | boolean| `bool` |  | | readOnly defaults to false (read/write). ReadOnly here will force</br>the ReadOnly setting in VolumeMounts.</br>+optional |  |
| secretRef | [LocalObjectReference](#local-object-reference)| `LocalObjectReference` |  | |  |  |
| volumeName | string| `string` |  | | volumeName is the human-readable name of the StorageOS volume.  Volume</br>names are only unique within a namespace. |  |
| volumeNamespace | string| `string` |  | | volumeNamespace specifies the scope of the volume within StorageOS.  If no</br>namespace is specified then the Pod's namespace will be used.  This allows the</br>Kubernetes name scoping to be mirrored within StorageOS for tighter integration.</br>Set VolumeName to any name to override the default behaviour.</br>Set to "default" if you are not using namespaces within StorageOS.</br>Namespaces that do not pre-exist within StorageOS will be created.</br>+optional |  |



### <span id="supplemental-groups-policy"></span> SupplementalGroupsPolicy


> SupplementalGroupsPolicy defines how supplemental groups
of the first container processes are calculated.
+enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| SupplementalGroupsPolicy | string| string | | SupplementalGroupsPolicy defines how supplemental groups</br>of the first container processes are calculated.</br>+enum |  |



### <span id="supplied-value-from"></span> SuppliedValueFrom


  

[interface{}](#interface)

### <span id="suspend-template"></span> SuspendTemplate


> SuspendTemplate is a template subtype to suspend a workflow at a predetermined point in time
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| duration | string| `string` |  | | Duration is the seconds to wait before automatically resuming a template. Must be a string. Default unit is seconds.</br>Could also be a Duration, e.g.: "2m", "6h" |  |



### <span id="synchronization"></span> Synchronization


> Synchronization holds synchronization lock configuration
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| mutex | [Mutex](#mutex)| `Mutex` |  | |  |  |
| mutexes | [][Mutex](#mutex)| `[]*Mutex` |  | | v3.6 and after: Mutexes holds the list of Mutex lock details |  |
| semaphore | [SemaphoreRef](#semaphore-ref)| `SemaphoreRef` |  | |  |  |
| semaphores | [][SemaphoreRef](#semaphore-ref)| `[]*SemaphoreRef` |  | | v3.6 and after: Semaphores holds the list of Semaphores configuration |  |



### <span id="sysctl"></span> Sysctl


> Sysctl defines a kernel parameter to be set
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| name | string| `string` |  | | Name of a property to set |  |
| value | string| `string` |  | | Value of a property to set |  |



### <span id="tcp-socket-action"></span> TCPSocketAction


> TCPSocketAction describes an action based on opening a socket
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| host | string| `string` |  | | Optional: Host name to connect to, defaults to the pod IP.</br>+optional |  |
| port | [IntOrString](#int-or-string)| `IntOrString` |  | |  |  |



### <span id="taint-effect"></span> TaintEffect


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| TaintEffect | string| string | | +enum |  |



### <span id="tar-strategy"></span> TarStrategy


> TarStrategy will tar and gzip the file or directory when saving
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| compressionLevel | int32 (formatted integer)| `int32` |  | | CompressionLevel specifies the gzip compression level to use for the artifact.</br>Defaults to gzip.DefaultCompression. |  |



### <span id="template"></span> Template


> Template is a reusable and composable unit of execution in a workflow
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| activeDeadlineSeconds | [IntOrString](#int-or-string)| `IntOrString` |  | |  |  |
| affinity | [Affinity](#affinity)| `Affinity` |  | |  |  |
| archiveLocation | [ArtifactLocation](#artifact-location)| `ArtifactLocation` |  | |  |  |
| automountServiceAccountToken | boolean| `bool` |  | | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods.</br>ServiceAccountName of ExecutorConfig must be specified if this value is false. |  |
| container | [Container](#container)| `Container` |  | |  |  |
| containerSet | [ContainerSetTemplate](#container-set-template)| `ContainerSetTemplate` |  | |  |  |
| daemon | boolean| `bool` |  | | Daemon will allow a workflow to proceed to the next step so long as the container reaches readiness |  |
| dag | [DAGTemplate](#d-a-g-template)| `DAGTemplate` |  | |  |  |
| data | [Data](#data)| `Data` |  | |  |  |
| executor | [ExecutorConfig](#executor-config)| `ExecutorConfig` |  | |  |  |
| failFast | boolean| `bool` |  | | FailFast, if specified, will fail this template if any of its child pods has failed. This is useful for when this</br>template is expanded with `withItems`, etc. |  |
| hostAliases | [][HostAlias](#host-alias)| `[]*HostAlias` |  | | HostAliases is an optional list of hosts and IPs that will be injected into the pod spec</br>+patchStrategy=merge</br>+patchMergeKey=ip |  |
| http | [HTTP](#http)| `HTTP` |  | |  |  |
| initContainers | [][UserContainer](#user-container)| `[]*UserContainer` |  | | InitContainers is a list of containers which run before the main container.</br>+patchStrategy=merge</br>+patchMergeKey=name |  |
| inputs | [Inputs](#inputs)| `Inputs` |  | |  |  |
| memoize | [Memoize](#memoize)| `Memoize` |  | |  |  |
| metadata | [Metadata](#metadata)| `Metadata` |  | |  |  |
| metrics | [Metrics](#metrics)| `Metrics` |  | |  |  |
| name | string| `string` |  | | Name is the name of the template |  |
| nodeSelector | map of string| `map[string]string` |  | | NodeSelector is a selector to schedule this step of the workflow to be</br>run on the selected node(s). Overrides the selector set at the workflow level. |  |
| outputs | [Outputs](#outputs)| `Outputs` |  | |  |  |
| parallelism | int64 (formatted integer)| `int64` |  | | Parallelism limits the max total parallel pods that can execute at the same time within the</br>boundaries of this template invocation. If additional steps/dag templates are invoked, the</br>pods created by those templates will not be counted towards this total. |  |
| plugin | [Plugin](#plugin)| `Plugin` |  | |  |  |
| podSpecPatch | string| `string` |  | | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of</br>container fields which are not strings (e.g. resource limits). |  |
| priority | int32 (formatted integer)| `int32` |  | | Priority to apply to workflow pods. |  |
| priorityClassName | string| `string` |  | | PriorityClassName to apply to workflow pods. |  |
| resource | [ResourceTemplate](#resource-template)| `ResourceTemplate` |  | |  |  |
| retryStrategy | [RetryStrategy](#retry-strategy)| `RetryStrategy` |  | |  |  |
| schedulerName | string| `string` |  | | If specified, the pod will be dispatched by specified scheduler.</br>Or it will be dispatched by workflow scope scheduler if specified.</br>If neither specified, the pod will be dispatched by default scheduler.</br>+optional |  |
| script | [ScriptTemplate](#script-template)| `ScriptTemplate` |  | |  |  |
| securityContext | [PodSecurityContext](#pod-security-context)| `PodSecurityContext` |  | |  |  |
| serviceAccountName | string| `string` |  | | ServiceAccountName to apply to workflow pods |  |
| sidecars | [][UserContainer](#user-container)| `[]*UserContainer` |  | | Sidecars is a list of containers which run alongside the main container</br>Sidecars are automatically killed when the main container completes</br>+patchStrategy=merge</br>+patchMergeKey=name |  |
| steps | [][ParallelSteps](#parallel-steps)| `[]ParallelSteps` |  | | Steps define a series of sequential/parallel workflow steps |  |
| suspend | [SuspendTemplate](#suspend-template)| `SuspendTemplate` |  | |  |  |
| synchronization | [Synchronization](#synchronization)| `Synchronization` |  | |  |  |
| timeout | string| `string` |  | | Timeout allows to set the total node execution timeout duration counting from the node's start time.</br>This duration also includes time in which the node spends in Pending state. This duration may not be applied to Step or DAG templates. |  |
| tolerations | [][Toleration](#toleration)| `[]*Toleration` |  | | Tolerations to apply to workflow pods.</br>+patchStrategy=merge</br>+patchMergeKey=key |  |
| volumes | [][Volume](#volume)| `[]*Volume` |  | | Volumes is a list of volumes that can be mounted by containers in a template.</br>+patchStrategy=merge</br>+patchMergeKey=name |  |



### <span id="template-ref"></span> TemplateRef


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| clusterScope | boolean| `bool` |  | | ClusterScope indicates the referred template is cluster scoped (i.e. a ClusterWorkflowTemplate). |  |
| name | string| `string` |  | | Name is the resource name of the template. |  |
| template | string| `string` |  | | Template is the name of referred template in the resource. |  |



### <span id="termination-message-policy"></span> TerminationMessagePolicy


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| TerminationMessagePolicy | string| string | | +enum |  |



### <span id="toleration"></span> Toleration


> The pod this Toleration is attached to tolerates any taint that matches
the triple <key,value,effect> using the matching operator <operator>.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| effect | [TaintEffect](#taint-effect)| `TaintEffect` |  | |  |  |
| key | string| `string` |  | | Key is the taint key that the toleration applies to. Empty means match all taint keys.</br>If the key is empty, operator must be Exists; this combination means to match all values and all keys.</br>+optional |  |
| operator | [TolerationOperator](#toleration-operator)| `TolerationOperator` |  | |  |  |
| tolerationSeconds | int64 (formatted integer)| `int64` |  | | TolerationSeconds represents the period of time the toleration (which must be</br>of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,</br>it is not set, which means tolerate the taint forever (do not evict). Zero and</br>negative values will be treated as 0 (evict immediately) by the system.</br>+optional |  |
| value | string| `string` |  | | Value is the taint value the toleration matches to.</br>If the operator is Exists, the value should be empty, otherwise just a regular string.</br>+optional |  |



### <span id="toleration-operator"></span> TolerationOperator


> +enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| TolerationOperator | string| string | | +enum |  |



### <span id="transformation"></span> Transformation


  

[][TransformationStep](#transformation-step)

### <span id="transformation-step"></span> TransformationStep


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| expression | string| `string` |  | | Expression defines an expr expression to apply |  |



### <span id="type"></span> Type


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| Type | int64 (formatted integer)| int64 | |  |  |



### <span id="typed-local-object-reference"></span> TypedLocalObjectReference


> New uses of this type are discouraged because of difficulty describing its usage when embedded in APIs.
1. Invalid usage help.  It is impossible to add specific help for individual usage.  In most embedded usages, there are particular
restrictions like, "must refer only to types A and B" or "UID not honored" or "name must be restricted".
Those cannot be well described when embedded.
2. Inconsistent validation.  Because the usages are different, the validation rules are different by usage, which makes it hard for users to predict what will happen.
3. The fields are both imprecise and overly precise.  Kind is not a precise mapping to a URL. This can produce ambiguity
during interpretation and require a REST mapping.  In most cases, the dependency is on the group,resource tuple
and the version of the actual struct is irrelevant.
4. We cannot easily change it.  Because this type is embedded in many locations, updates to this type
will affect numerous schemas.  Don't make new APIs embed an underspecified API type they do not control.

Instead of using this type, create a locally provided and used type that is well-focused on your reference.
For example, ServiceReferences for admission registration: https://github.com/kubernetes/api/blob/release-1.17/admissionregistration/v1/types.go#L533 .
+structType=atomic
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| apiGroup | string| `string` |  | | APIGroup is the group for the resource being referenced.</br>If APIGroup is not specified, the specified Kind must be in the core API group.</br>For any other third-party types, APIGroup is required.</br>+optional |  |
| kind | string| `string` |  | | Kind is the type of resource being referenced |  |
| name | string| `string` |  | | Name is the name of resource being referenced |  |



### <span id="typed-object-reference"></span> TypedObjectReference


> TypedObjectReference contains enough information to let you locate the typed referenced object
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| apiGroup | string| `string` |  | | APIGroup is the group for the resource being referenced.</br>If APIGroup is not specified, the specified Kind must be in the core API group.</br>For any other third-party types, APIGroup is required.</br>+optional |  |
| kind | string| `string` |  | | Kind is the type of resource being referenced |  |
| name | string| `string` |  | | Name is the name of resource being referenced |  |
| namespace | string| `string` |  | | Namespace is the namespace of resource being referenced</br>Note that when a namespace is specified, a gateway.networking.k8s.io/ReferenceGrant object is required in the referent namespace to allow that namespace's owner to accept the reference. See the ReferenceGrant documentation for details.</br>(Alpha) This field requires the CrossNamespaceVolumeDataSource feature gate to be enabled.</br>+featureGate=CrossNamespaceVolumeDataSource</br>+optional |  |



### <span id="uid"></span> UID


> UID is a type that holds unique ID values, including UUIDs.  Because we
don't ONLY use UUIDs, this is an alias to string.  Being a type captures
intent and helps make sure that UIDs and names do not get conflated.
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| UID | string| string | | UID is a type that holds unique ID values, including UUIDs.  Because we</br>don't ONLY use UUIDs, this is an alias to string.  Being a type captures</br>intent and helps make sure that UIDs and names do not get conflated. |  |



### <span id="uri-scheme"></span> URIScheme


> URIScheme identifies the scheme used for connection to a host for Get actions
+enum
  



| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| URIScheme | string| string | | URIScheme identifies the scheme used for connection to a host for Get actions</br>+enum |  |



### <span id="user-container"></span> UserContainer


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| args | []string| `[]string` |  | | Arguments to the entrypoint.</br>The container image's CMD is used if this is not provided.</br>Variable references $(VAR_NAME) are expanded using the container's environment. If a variable</br>cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced</br>to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will</br>produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless</br>of whether the variable exists or not. Cannot be updated.</br>More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell</br>+optional</br>+listType=atomic |  |
| command | []string| `[]string` |  | | Entrypoint array. Not executed within a shell.</br>The container image's ENTRYPOINT is used if this is not provided.</br>Variable references $(VAR_NAME) are expanded using the container's environment. If a variable</br>cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced</br>to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will</br>produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless</br>of whether the variable exists or not. Cannot be updated.</br>More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell</br>+optional</br>+listType=atomic |  |
| env | [][EnvVar](#env-var)| `[]*EnvVar` |  | | List of environment variables to set in the container.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=name</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=name |  |
| envFrom | [][EnvFromSource](#env-from-source)| `[]*EnvFromSource` |  | | List of sources to populate environment variables in the container.</br>The keys defined within a source must be a C_IDENTIFIER. All invalid keys</br>will be reported as an event when the container is starting. When a key exists in multiple</br>sources, the value associated with the last source will take precedence.</br>Values defined by an Env with a duplicate key will take precedence.</br>Cannot be updated.</br>+optional</br>+listType=atomic |  |
| image | string| `string` |  | | Container image name.</br>More info: https://kubernetes.io/docs/concepts/containers/images</br>This field is optional to allow higher level config management to default or override</br>container images in workload controllers like Deployments and StatefulSets.</br>+optional |  |
| imagePullPolicy | [PullPolicy](#pull-policy)| `PullPolicy` |  | |  |  |
| lifecycle | [Lifecycle](#lifecycle)| `Lifecycle` |  | |  |  |
| livenessProbe | [Probe](#probe)| `Probe` |  | |  |  |
| mirrorVolumeMounts | boolean| `bool` |  | | MirrorVolumeMounts will mount the same volumes specified in the main container</br>to the container (including artifacts), at the same mountPaths. This enables</br>dind daemon to partially see the same filesystem as the main container in</br>order to use features such as docker volume binding |  |
| name | string| `string` |  | | Name of the container specified as a DNS_LABEL.</br>Each container in a pod must have a unique name (DNS_LABEL).</br>Cannot be updated. |  |
| ports | [][ContainerPort](#container-port)| `[]*ContainerPort` |  | | List of ports to expose from the container. Not specifying a port here</br>DOES NOT prevent that port from being exposed. Any port which is</br>listening on the default "0.0.0.0" address inside a container will be</br>accessible from the network.</br>Modifying this array with strategic merge patch may corrupt the data.</br>For more information See https://github.com/kubernetes/kubernetes/issues/108255.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=containerPort</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=containerPort</br>+listMapKey=protocol |  |
| readinessProbe | [Probe](#probe)| `Probe` |  | |  |  |
| resizePolicy | [][ContainerResizePolicy](#container-resize-policy)| `[]*ContainerResizePolicy` |  | | Resources resize policy for the container.</br>+featureGate=InPlacePodVerticalScaling</br>+optional</br>+listType=atomic |  |
| resources | [ResourceRequirements](#resource-requirements)| `ResourceRequirements` |  | |  |  |
| restartPolicy | [ContainerRestartPolicy](#container-restart-policy)| `ContainerRestartPolicy` |  | |  |  |
| securityContext | [SecurityContext](#security-context)| `SecurityContext` |  | |  |  |
| startupProbe | [Probe](#probe)| `Probe` |  | |  |  |
| stdin | boolean| `bool` |  | | Whether this container should allocate a buffer for stdin in the container runtime. If this</br>is not set, reads from stdin in the container will always result in EOF.</br>Default is false.</br>+optional |  |
| stdinOnce | boolean| `bool` |  | | Whether the container runtime should close the stdin channel after it has been opened by</br>a single attach. When stdin is true the stdin stream will remain open across multiple attach</br>sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the</br>first client attaches to stdin, and then remains open and accepts data until the client disconnects,</br>at which time stdin is closed and remains closed until the container is restarted. If this</br>flag is false, a container processes that reads from stdin will never receive an EOF.</br>Default is false</br>+optional |  |
| terminationMessagePath | string| `string` |  | | Optional: Path at which the file to which the container's termination message</br>will be written is mounted into the container's filesystem.</br>Message written is intended to be brief final status, such as an assertion failure message.</br>Will be truncated by the node if greater than 4096 bytes. The total message length across</br>all containers will be limited to 12kb.</br>Defaults to /dev/termination-log.</br>Cannot be updated.</br>+optional |  |
| terminationMessagePolicy | [TerminationMessagePolicy](#termination-message-policy)| `TerminationMessagePolicy` |  | |  |  |
| tty | boolean| `bool` |  | | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true.</br>Default is false.</br>+optional |  |
| volumeDevices | [][VolumeDevice](#volume-device)| `[]*VolumeDevice` |  | | volumeDevices is the list of block devices to be used by the container.</br>+patchMergeKey=devicePath</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=devicePath</br>+optional |  |
| volumeMounts | [][VolumeMount](#volume-mount)| `[]*VolumeMount` |  | | Pod volumes to mount into the container's filesystem.</br>Cannot be updated.</br>+optional</br>+patchMergeKey=mountPath</br>+patchStrategy=merge</br>+listType=map</br>+listMapKey=mountPath |  |
| workingDir | string| `string` |  | | Container's working directory.</br>If not specified, the container runtime's default will be used, which</br>might be configured in the container image.</br>Cannot be updated.</br>+optional |  |



### <span id="value-from"></span> ValueFrom


> ValueFrom describes a location in which to obtain the value to a parameter
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| configMapKeyRef | [ConfigMapKeySelector](#config-map-key-selector)| `ConfigMapKeySelector` |  | |  |  |
| default | [AnyString](#any-string)| `AnyString` |  | |  |  |
| event | string| `string` |  | | Selector (https://github.com/expr-lang/expr) that is evaluated against the event to get the value of the parameter. E.g. `payload.message` |  |
| expression | string| `string` |  | | Expression, if defined, is evaluated to specify the value for the parameter |  |
| jqFilter | string| `string` |  | | JQFilter expression against the resource object in resource templates |  |
| jsonPath | string| `string` |  | | JSONPath of a resource to retrieve an output parameter value from in resource templates |  |
| parameter | string| `string` |  | | Parameter reference to a step or dag task in which to retrieve an output parameter value from</br>(e.g. '{{steps.mystep.outputs.myparam}}') |  |
| path | string| `string` |  | | Path in the container to retrieve an output parameter value from in container templates |  |
| supplied | [SuppliedValueFrom](#supplied-value-from)| `SuppliedValueFrom` |  | |  |  |



### <span id="volume"></span> Volume


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| awsElasticBlockStore | [AWSElasticBlockStoreVolumeSource](#a-w-s-elastic-block-store-volume-source)| `AWSElasticBlockStoreVolumeSource` |  | |  |  |
| azureDisk | [AzureDiskVolumeSource](#azure-disk-volume-source)| `AzureDiskVolumeSource` |  | |  |  |
| azureFile | [AzureFileVolumeSource](#azure-file-volume-source)| `AzureFileVolumeSource` |  | |  |  |
| cephfs | [CephFSVolumeSource](#ceph-f-s-volume-source)| `CephFSVolumeSource` |  | |  |  |
| cinder | [CinderVolumeSource](#cinder-volume-source)| `CinderVolumeSource` |  | |  |  |
| configMap | [ConfigMapVolumeSource](#config-map-volume-source)| `ConfigMapVolumeSource` |  | |  |  |
| csi | [CSIVolumeSource](#c-s-i-volume-source)| `CSIVolumeSource` |  | |  |  |
| downwardAPI | [DownwardAPIVolumeSource](#downward-api-volume-source)| `DownwardAPIVolumeSource` |  | |  |  |
| emptyDir | [EmptyDirVolumeSource](#empty-dir-volume-source)| `EmptyDirVolumeSource` |  | |  |  |
| ephemeral | [EphemeralVolumeSource](#ephemeral-volume-source)| `EphemeralVolumeSource` |  | |  |  |
| fc | [FCVolumeSource](#f-c-volume-source)| `FCVolumeSource` |  | |  |  |
| flexVolume | [FlexVolumeSource](#flex-volume-source)| `FlexVolumeSource` |  | |  |  |
| flocker | [FlockerVolumeSource](#flocker-volume-source)| `FlockerVolumeSource` |  | |  |  |
| gcePersistentDisk | [GCEPersistentDiskVolumeSource](#g-c-e-persistent-disk-volume-source)| `GCEPersistentDiskVolumeSource` |  | |  |  |
| gitRepo | [GitRepoVolumeSource](#git-repo-volume-source)| `GitRepoVolumeSource` |  | |  |  |
| glusterfs | [GlusterfsVolumeSource](#glusterfs-volume-source)| `GlusterfsVolumeSource` |  | |  |  |
| hostPath | [HostPathVolumeSource](#host-path-volume-source)| `HostPathVolumeSource` |  | |  |  |
| image | [ImageVolumeSource](#image-volume-source)| `ImageVolumeSource` |  | |  |  |
| iscsi | [ISCSIVolumeSource](#i-s-c-s-i-volume-source)| `ISCSIVolumeSource` |  | |  |  |
| name | string| `string` |  | | name of the volume.</br>Must be a DNS_LABEL and unique within the pod.</br>More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names |  |
| nfs | [NFSVolumeSource](#n-f-s-volume-source)| `NFSVolumeSource` |  | |  |  |
| persistentVolumeClaim | [PersistentVolumeClaimVolumeSource](#persistent-volume-claim-volume-source)| `PersistentVolumeClaimVolumeSource` |  | |  |  |
| photonPersistentDisk | [PhotonPersistentDiskVolumeSource](#photon-persistent-disk-volume-source)| `PhotonPersistentDiskVolumeSource` |  | |  |  |
| portworxVolume | [PortworxVolumeSource](#portworx-volume-source)| `PortworxVolumeSource` |  | |  |  |
| projected | [ProjectedVolumeSource](#projected-volume-source)| `ProjectedVolumeSource` |  | |  |  |
| quobyte | [QuobyteVolumeSource](#quobyte-volume-source)| `QuobyteVolumeSource` |  | |  |  |
| rbd | [RBDVolumeSource](#r-b-d-volume-source)| `RBDVolumeSource` |  | |  |  |
| scaleIO | [ScaleIOVolumeSource](#scale-i-o-volume-source)| `ScaleIOVolumeSource` |  | |  |  |
| secret | [SecretVolumeSource](#secret-volume-source)| `SecretVolumeSource` |  | |  |  |
| storageos | [StorageOSVolumeSource](#storage-o-s-volume-source)| `StorageOSVolumeSource` |  | |  |  |
| vsphereVolume | [VsphereVirtualDiskVolumeSource](#vsphere-virtual-disk-volume-source)| `VsphereVirtualDiskVolumeSource` |  | |  |  |



### <span id="volume-device"></span> VolumeDevice


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| devicePath | string| `string` |  | | devicePath is the path inside of the container that the device will be mapped to. |  |
| name | string| `string` |  | | name must match the name of a persistentVolumeClaim in the pod |  |



### <span id="volume-mount"></span> VolumeMount


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| mountPath | string| `string` |  | | Path within the container at which the volume should be mounted.  Must</br>not contain ':'. |  |
| mountPropagation | [MountPropagationMode](#mount-propagation-mode)| `MountPropagationMode` |  | |  |  |
| name | string| `string` |  | | This must match the Name of a Volume. |  |
| readOnly | boolean| `bool` |  | | Mounted read-only if true, read-write otherwise (false or unspecified).</br>Defaults to false.</br>+optional |  |
| recursiveReadOnly | [RecursiveReadOnlyMode](#recursive-read-only-mode)| `RecursiveReadOnlyMode` |  | |  |  |
| subPath | string| `string` |  | | Path within the volume from which the container's volume should be mounted.</br>Defaults to "" (volume's root).</br>+optional |  |
| subPathExpr | string| `string` |  | | Expanded path within the volume from which the container's volume should be mounted.</br>Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment.</br>Defaults to "" (volume's root).</br>SubPathExpr and SubPath are mutually exclusive.</br>+optional |  |



### <span id="volume-projection"></span> VolumeProjection


> Exactly one of these fields must be set.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| clusterTrustBundle | [ClusterTrustBundleProjection](#cluster-trust-bundle-projection)| `ClusterTrustBundleProjection` |  | |  |  |
| configMap | [ConfigMapProjection](#config-map-projection)| `ConfigMapProjection` |  | |  |  |
| downwardAPI | [DownwardAPIProjection](#downward-api-projection)| `DownwardAPIProjection` |  | |  |  |
| secret | [SecretProjection](#secret-projection)| `SecretProjection` |  | |  |  |
| serviceAccountToken | [ServiceAccountTokenProjection](#service-account-token-projection)| `ServiceAccountTokenProjection` |  | |  |  |



### <span id="volume-resource-requirements"></span> VolumeResourceRequirements


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| limits | [ResourceList](#resource-list)| `ResourceList` |  | |  |  |
| requests | [ResourceList](#resource-list)| `ResourceList` |  | |  |  |



### <span id="vsphere-virtual-disk-volume-source"></span> VsphereVirtualDiskVolumeSource


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| fsType | string| `string` |  | | fsType is filesystem type to mount.</br>Must be a filesystem type supported by the host operating system.</br>Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.</br>+optional |  |
| storagePolicyID | string| `string` |  | | storagePolicyID is the storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName.</br>+optional |  |
| storagePolicyName | string| `string` |  | | storagePolicyName is the storage Policy Based Management (SPBM) profile name.</br>+optional |  |
| volumePath | string| `string` |  | | volumePath is the path that identifies vSphere volume vmdk |  |



### <span id="weighted-pod-affinity-term"></span> WeightedPodAffinityTerm


> The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| podAffinityTerm | [PodAffinityTerm](#pod-affinity-term)| `PodAffinityTerm` |  | |  |  |
| weight | int32 (formatted integer)| `int32` |  | | weight associated with matching the corresponding podAffinityTerm,</br>in the range 1-100. |  |



### <span id="windows-security-context-options"></span> WindowsSecurityContextOptions


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| gmsaCredentialSpec | string| `string` |  | | GMSACredentialSpec is where the GMSA admission webhook</br>(https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the</br>GMSA credential spec named by the GMSACredentialSpecName field.</br>+optional |  |
| gmsaCredentialSpecName | string| `string` |  | | GMSACredentialSpecName is the name of the GMSA credential spec to use.</br>+optional |  |
| hostProcess | boolean| `bool` |  | | HostProcess determines if a container should be run as a 'Host Process' container.</br>All of a Pod's containers must have the same effective HostProcess value</br>(it is not allowed to have a mix of HostProcess containers and non-HostProcess containers).</br>In addition, if HostProcess is true then HostNetwork must also be set to true.</br>+optional |  |
| runAsUserName | string| `string` |  | | The UserName in Windows to run the entrypoint of the container process.</br>Defaults to the user specified in image metadata if unspecified.</br>May also be set in PodSecurityContext. If set in both SecurityContext and</br>PodSecurityContext, the value specified in SecurityContext takes precedence.</br>+optional |  |



### <span id="workflow"></span> Workflow


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| metadata | [ObjectMeta](#object-meta)| `ObjectMeta` | ✓ | |  |  |



### <span id="zip-strategy"></span> ZipStrategy


> ZipStrategy will unzip zipped input artifacts
  



[interface{}](#interface)
