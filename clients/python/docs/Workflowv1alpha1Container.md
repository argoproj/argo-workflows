# Workflowv1alpha1Container

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**args** | **list[str]** |  | [optional] 
**command** | **list[str]** |  | [optional] 
**env** | [**list[Workflowv1alpha1EnvVar]**](Workflowv1alpha1EnvVar.md) |  | [optional] 
**env_from** | [**list[Workflowv1alpha1EnvFromSource]**](Workflowv1alpha1EnvFromSource.md) |  | [optional] 
**image** | **str** |  | [optional] 
**image_pull_policy** | **str** |  | [optional] 
**lifecycle** | [**Workflowv1alpha1Lifecycle**](Workflowv1alpha1Lifecycle.md) |  | [optional] 
**liveness_probe** | [**Workflowv1alpha1Probe**](Workflowv1alpha1Probe.md) |  | [optional] 
**name** | **str** | Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated. | [optional] 
**ports** | [**list[Workflowv1alpha1ContainerPort]**](Workflowv1alpha1ContainerPort.md) |  | [optional] 
**readiness_probe** | [**Workflowv1alpha1Probe**](Workflowv1alpha1Probe.md) |  | [optional] 
**resources** | [**Workflowv1alpha1ResourceRequirements**](Workflowv1alpha1ResourceRequirements.md) |  | [optional] 
**security_context** | [**Workflowv1alpha1SecurityContext**](Workflowv1alpha1SecurityContext.md) |  | [optional] 
**startup_probe** | [**Workflowv1alpha1Probe**](Workflowv1alpha1Probe.md) |  | [optional] 
**stdin** | **bool** |  | [optional] 
**stdin_once** | **bool** |  | [optional] 
**termination_message_path** | **str** |  | [optional] 
**termination_message_policy** | **str** |  | [optional] 
**tty** | **bool** |  | [optional] 
**volume_devices** | [**list[Workflowv1alpha1VolumeDevice]**](Workflowv1alpha1VolumeDevice.md) |  | [optional] 
**volume_mounts** | [**list[Workflowv1alpha1VolumeMount]**](Workflowv1alpha1VolumeMount.md) |  | [optional] 
**working_dir** | **str** |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


