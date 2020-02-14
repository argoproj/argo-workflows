# V1Container

A single application container that you want to run within a pod.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**args** | **list[str]** |  | [optional] 
**command** | **list[str]** |  | [optional] 
**env** | [**list[V1EnvVar]**](V1EnvVar.md) |  | [optional] 
**env_from** | [**list[V1EnvFromSource]**](V1EnvFromSource.md) |  | [optional] 
**image** | **str** |  | [optional] 
**image_pull_policy** | **str** |  | [optional] 
**lifecycle** | [**V1Lifecycle**](V1Lifecycle.md) |  | [optional] 
**liveness_probe** | [**V1Probe**](V1Probe.md) |  | [optional] 
**name** | **str** | Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated. | [optional] 
**ports** | [**list[V1ContainerPort]**](V1ContainerPort.md) |  | [optional] 
**readiness_probe** | [**V1Probe**](V1Probe.md) |  | [optional] 
**resources** | [**V1ResourceRequirements**](V1ResourceRequirements.md) |  | [optional] 
**security_context** | [**V1SecurityContext**](V1SecurityContext.md) |  | [optional] 
**startup_probe** | [**V1Probe**](V1Probe.md) |  | [optional] 
**stdin** | **bool** |  | [optional] 
**stdin_once** | **bool** |  | [optional] 
**termination_message_path** | **str** |  | [optional] 
**termination_message_policy** | **str** |  | [optional] 
**tty** | **bool** |  | [optional] 
**volume_devices** | [**list[V1VolumeDevice]**](V1VolumeDevice.md) |  | [optional] 
**volume_mounts** | [**list[V1VolumeMount]**](V1VolumeMount.md) |  | [optional] 
**working_dir** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


