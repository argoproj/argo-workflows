# AWSElasticBlockStoreVolumeSource

Represents a Persistent Disk resource in AWS.  An AWS EBS disk must exist before mounting to a container. The disk must also be in the same AWS zone as the kubelet. An AWS EBS disk can only be mounted as read/write once. AWS EBS volumes support ownership management and SELinux relabeling.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_type** | **str** | Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \&quot;ext4\&quot;, \&quot;xfs\&quot;, \&quot;ntfs\&quot;. Implicitly inferred to be \&quot;ext4\&quot; if unspecified. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore | [optional] 
**partition** | **int** | The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as \&quot;1\&quot;. Similarly, the volume partition for /dev/sda is \&quot;0\&quot; (or you can leave the property empty). | [optional] 
**read_only** | **bool** | Specify \&quot;true\&quot; to force and set the ReadOnly property in VolumeMounts to \&quot;true\&quot;. If omitted, the default is \&quot;false\&quot;. More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore | [optional] 
**volume_id** | **str** | Unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore | 

## Example

```python
from argo_workflows.models.aws_elastic_block_store_volume_source import AWSElasticBlockStoreVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of AWSElasticBlockStoreVolumeSource from a JSON string
aws_elastic_block_store_volume_source_instance = AWSElasticBlockStoreVolumeSource.from_json(json)
# print the JSON string representation of the object
print(AWSElasticBlockStoreVolumeSource.to_json())

# convert the object into a dict
aws_elastic_block_store_volume_source_dict = aws_elastic_block_store_volume_source_instance.to_dict()
# create an instance of AWSElasticBlockStoreVolumeSource from a dict
aws_elastic_block_store_volume_source_form_dict = aws_elastic_block_store_volume_source.from_dict(aws_elastic_block_store_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


