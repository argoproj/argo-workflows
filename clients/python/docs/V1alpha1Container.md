# V1alpha1Container

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**args** | **list[str]** |  | [optional] 
**command** | **list[str]** |  | [optional] 
**env** | [**list[V1alpha1EnvVar]**](V1alpha1EnvVar.md) |  | [optional] 
**env_from** | [**list[V1alpha1EnvFromSource]**](V1alpha1EnvFromSource.md) |  | [optional] 
**image** | **str** |  | [optional] 
**image_pull_policy** | **str** |  | [optional] 
**lifecycle** | [**V1alpha1Lifecycle**](V1alpha1Lifecycle.md) |  | [optional] 
**liveness_probe** | [**V1alpha1Probe**](V1alpha1Probe.md) |  | [optional] 
**name** | **str** | Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated. | [optional] 
**ports** | [**list[V1alpha1ContainerPort]**](V1alpha1ContainerPort.md) |  | [optional] 
**readiness_probe** | [**V1alpha1Probe**](V1alpha1Probe.md) |  | [optional] 
**resources** | [**V1alpha1ResourceRequirements**](V1alpha1ResourceRequirements.md) |  | [optional] 
**security_context** | [**V1alpha1SecurityContext**](V1alpha1SecurityContext.md) |  | [optional] 
**startup_probe** | [**V1alpha1Probe**](V1alpha1Probe.md) |  | [optional] 
**stdin** | **bool** |  | [optional] 
**stdin_once** | **bool** |  | [optional] 
**termination_message_path** | **str** |  | [optional] 
**termination_message_policy** | **str** |  | [optional] 
**tty** | **bool** |  | [optional] 
**volume_devices** | [**list[V1alpha1VolumeDevice]**](V1alpha1VolumeDevice.md) |  | [optional] 
**volume_mounts** | [**list[V1alpha1VolumeMount]**](V1alpha1VolumeMount.md) |  | [optional] 
**working_dir** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


