

# ContainerStatus

ContainerStatus contains details for the current status of this container.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**allocatedResources** | **Map&lt;String, String&gt;** | AllocatedResources represents the compute resources allocated for this container by the node. Kubelet sets this value to Container.Resources.Requests upon successful pod admission and after successfully admitting desired pod resize. |  [optional]
**allocatedResourcesStatus** | [**List&lt;ResourceStatus&gt;**](ResourceStatus.md) | AllocatedResourcesStatus represents the status of various resources allocated for this Pod. |  [optional]
**containerID** | **String** | ContainerID is the ID of the container in the format &#39;&lt;type&gt;://&lt;container_id&gt;&#39;. Where type is a container runtime identifier, returned from Version call of CRI API (for example \&quot;containerd\&quot;). |  [optional]
**image** | **String** | Image is the name of container image that the container is running. The container image may not match the image used in the PodSpec, as it may have been resolved by the runtime. More info: https://kubernetes.io/docs/concepts/containers/images. | 
**imageID** | **String** | ImageID is the image ID of the container&#39;s image. The image ID may not match the image ID of the image used in the PodSpec, as it may have been resolved by the runtime. | 
**lastState** | [**ContainerState**](ContainerState.md) |  |  [optional]
**name** | **String** | Name is a DNS_LABEL representing the unique name of the container. Each container in a pod must have a unique name across all container types. Cannot be updated. | 
**ready** | **Boolean** | Ready specifies whether the container is currently passing its readiness check. The value will change as readiness probes keep executing. If no readiness probes are specified, this field defaults to true once the container is fully started (see Started field).  The value is typically used to determine whether a container is ready to accept traffic. | 
**resources** | [**io.kubernetes.client.openapi.models.V1ResourceRequirements**](io.kubernetes.client.openapi.models.V1ResourceRequirements.md) |  |  [optional]
**restartCount** | **Integer** | RestartCount holds the number of times the container has been restarted. Kubelet makes an effort to always increment the value, but there are cases when the state may be lost due to node restarts and then the value may be reset to 0. The value is never negative. | 
**started** | **Boolean** | Started indicates whether the container has finished its postStart lifecycle hook and passed its startup probe. Initialized as false, becomes true after startupProbe is considered successful. Resets to false when the container is restarted, or if kubelet loses state temporarily. In both cases, startup probes will run again. Is always true when no startupProbe is defined and container is running and has passed the postStart lifecycle hook. The null value must be treated the same as false. |  [optional]
**state** | [**ContainerState**](ContainerState.md) |  |  [optional]
**stopSignal** | **String** | StopSignal reports the effective stop signal for this container |  [optional]
**user** | [**ContainerUser**](ContainerUser.md) |  |  [optional]
**volumeMounts** | [**List&lt;VolumeMountStatus&gt;**](VolumeMountStatus.md) | Status of volume mounts. |  [optional]



