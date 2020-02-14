

# V1Container

A single application container that you want to run within a pod.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**args** | **List&lt;String&gt;** |  |  [optional]
**command** | **List&lt;String&gt;** |  |  [optional]
**env** | [**List&lt;V1EnvVar&gt;**](V1EnvVar.md) |  |  [optional]
**envFrom** | [**List&lt;V1EnvFromSource&gt;**](V1EnvFromSource.md) |  |  [optional]
**image** | **String** |  |  [optional]
**imagePullPolicy** | **String** |  |  [optional]
**lifecycle** | [**V1Lifecycle**](V1Lifecycle.md) |  |  [optional]
**livenessProbe** | [**V1Probe**](V1Probe.md) |  |  [optional]
**name** | **String** | Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated. |  [optional]
**ports** | [**List&lt;V1ContainerPort&gt;**](V1ContainerPort.md) |  |  [optional]
**readinessProbe** | [**V1Probe**](V1Probe.md) |  |  [optional]
**resources** | [**V1ResourceRequirements**](V1ResourceRequirements.md) |  |  [optional]
**securityContext** | [**V1SecurityContext**](V1SecurityContext.md) |  |  [optional]
**startupProbe** | [**V1Probe**](V1Probe.md) |  |  [optional]
**stdin** | **Boolean** |  |  [optional]
**stdinOnce** | **Boolean** |  |  [optional]
**terminationMessagePath** | **String** |  |  [optional]
**terminationMessagePolicy** | **String** |  |  [optional]
**tty** | **Boolean** |  |  [optional]
**volumeDevices** | [**List&lt;V1VolumeDevice&gt;**](V1VolumeDevice.md) |  |  [optional]
**volumeMounts** | [**List&lt;V1VolumeMount&gt;**](V1VolumeMount.md) |  |  [optional]
**workingDir** | **String** |  |  [optional]



