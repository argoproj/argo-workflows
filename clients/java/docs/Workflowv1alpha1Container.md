

# Workflowv1alpha1Container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**args** | **List&lt;String&gt;** |  |  [optional]
**command** | **List&lt;String&gt;** |  |  [optional]
**env** | [**List&lt;Workflowv1alpha1EnvVar&gt;**](Workflowv1alpha1EnvVar.md) |  |  [optional]
**envFrom** | [**List&lt;Workflowv1alpha1EnvFromSource&gt;**](Workflowv1alpha1EnvFromSource.md) |  |  [optional]
**image** | **String** |  |  [optional]
**imagePullPolicy** | **String** |  |  [optional]
**lifecycle** | [**Workflowv1alpha1Lifecycle**](Workflowv1alpha1Lifecycle.md) |  |  [optional]
**livenessProbe** | [**Workflowv1alpha1Probe**](Workflowv1alpha1Probe.md) |  |  [optional]
**name** | **String** | Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated. |  [optional]
**ports** | [**List&lt;Workflowv1alpha1ContainerPort&gt;**](Workflowv1alpha1ContainerPort.md) |  |  [optional]
**readinessProbe** | [**Workflowv1alpha1Probe**](Workflowv1alpha1Probe.md) |  |  [optional]
**resources** | [**Workflowv1alpha1ResourceRequirements**](Workflowv1alpha1ResourceRequirements.md) |  |  [optional]
**securityContext** | [**Workflowv1alpha1SecurityContext**](Workflowv1alpha1SecurityContext.md) |  |  [optional]
**startupProbe** | [**Workflowv1alpha1Probe**](Workflowv1alpha1Probe.md) |  |  [optional]
**stdin** | **Boolean** |  |  [optional]
**stdinOnce** | **Boolean** |  |  [optional]
**terminationMessagePath** | **String** |  |  [optional]
**terminationMessagePolicy** | **String** |  |  [optional]
**tty** | **Boolean** |  |  [optional]
**volumeDevices** | [**List&lt;Workflowv1alpha1VolumeDevice&gt;**](Workflowv1alpha1VolumeDevice.md) |  |  [optional]
**volumeMounts** | [**List&lt;Workflowv1alpha1VolumeMount&gt;**](Workflowv1alpha1VolumeMount.md) |  |  [optional]
**workingDir** | **String** |  |  [optional]



