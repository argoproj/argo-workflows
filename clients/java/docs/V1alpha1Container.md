

# V1alpha1Container

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**args** | **List&lt;String&gt;** |  |  [optional]
**command** | **List&lt;String&gt;** |  |  [optional]
**env** | [**List&lt;V1alpha1EnvVar&gt;**](V1alpha1EnvVar.md) |  |  [optional]
**envFrom** | [**List&lt;V1alpha1EnvFromSource&gt;**](V1alpha1EnvFromSource.md) |  |  [optional]
**image** | **String** |  |  [optional]
**imagePullPolicy** | **String** |  |  [optional]
**lifecycle** | [**V1alpha1Lifecycle**](V1alpha1Lifecycle.md) |  |  [optional]
**livenessProbe** | [**V1alpha1Probe**](V1alpha1Probe.md) |  |  [optional]
**name** | **String** | Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated. |  [optional]
**ports** | [**List&lt;V1alpha1ContainerPort&gt;**](V1alpha1ContainerPort.md) |  |  [optional]
**readinessProbe** | [**V1alpha1Probe**](V1alpha1Probe.md) |  |  [optional]
**resources** | [**V1alpha1ResourceRequirements**](V1alpha1ResourceRequirements.md) |  |  [optional]
**securityContext** | [**V1alpha1SecurityContext**](V1alpha1SecurityContext.md) |  |  [optional]
**startupProbe** | [**V1alpha1Probe**](V1alpha1Probe.md) |  |  [optional]
**stdin** | **Boolean** |  |  [optional]
**stdinOnce** | **Boolean** |  |  [optional]
**terminationMessagePath** | **String** |  |  [optional]
**terminationMessagePolicy** | **String** |  |  [optional]
**tty** | **Boolean** |  |  [optional]
**volumeDevices** | [**List&lt;V1alpha1VolumeDevice&gt;**](V1alpha1VolumeDevice.md) |  |  [optional]
**volumeMounts** | [**List&lt;V1alpha1VolumeMount&gt;**](V1alpha1VolumeMount.md) |  |  [optional]
**workingDir** | **String** |  |  [optional]



