

# V1alpha1WorkflowSpec

WorkflowSpec is the specification of a Workflow.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**activeDeadlineSeconds** | **String** |  |  [optional]
**affinity** | [**V1Affinity**](V1Affinity.md) |  |  [optional]
**arguments** | [**V1alpha1Arguments**](V1alpha1Arguments.md) |  |  [optional]
**artifactRepositoryRef** | [**V1alpha1ArtifactRepositoryRef**](V1alpha1ArtifactRepositoryRef.md) |  |  [optional]
**automountServiceAccountToken** | **Boolean** | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods. ServiceAccountName of ExecutorConfig must be specified if this value is false. |  [optional]
**dnsConfig** | [**V1PodDNSConfig**](V1PodDNSConfig.md) |  |  [optional]
**dnsPolicy** | **String** | Set DNS policy for the pod. Defaults to \&quot;ClusterFirst\&quot;. Valid values are &#39;ClusterFirstWithHostNet&#39;, &#39;ClusterFirst&#39;, &#39;Default&#39; or &#39;None&#39;. DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy. To have DNS options set along with hostNetwork, you have to specify DNS policy explicitly to &#39;ClusterFirstWithHostNet&#39;. |  [optional]
**entrypoint** | **String** |  |  [optional]
**executor** | [**V1alpha1ExecutorConfig**](V1alpha1ExecutorConfig.md) |  |  [optional]
**hostAliases** | [**List&lt;V1HostAlias&gt;**](V1HostAlias.md) |  |  [optional]
**hostNetwork** | **Boolean** | Host networking requested for this workflow pod. Default to false. |  [optional]
**imagePullSecrets** | [**List&lt;V1LocalObjectReference&gt;**](V1LocalObjectReference.md) |  |  [optional]
**nodeSelector** | **Map&lt;String, String&gt;** | NodeSelector is a selector which will result in all pods of the workflow to be scheduled on the selected node(s). This is able to be overridden by a nodeSelector specified in the template. |  [optional]
**onExit** | **String** | OnExit is a template reference which is invoked at the end of the workflow, irrespective of the success, failure, or error of the primary workflow. |  [optional]
**parallelism** | **String** |  |  [optional]
**podGC** | [**V1alpha1PodGC**](V1alpha1PodGC.md) |  |  [optional]
**podPriority** | **Integer** | Priority to apply to workflow pods. |  [optional]
**podPriorityClassName** | **String** | PriorityClassName to apply to workflow pods. |  [optional]
**podSpecPatch** | **String** | PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization of container fields which are not strings (e.g. resource limits). |  [optional]
**priority** | **Integer** | Priority is used if controller is configured to process limited number of workflows in parallel. Workflows with higher priority are processed first. |  [optional]
**schedulerName** | **String** |  |  [optional]
**securityContext** | [**V1PodSecurityContext**](V1PodSecurityContext.md) |  |  [optional]
**serviceAccountName** | **String** | ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as. |  [optional]
**suspend** | **Boolean** |  |  [optional]
**templates** | [**List&lt;V1alpha1Template&gt;**](V1alpha1Template.md) |  |  [optional]
**tolerations** | [**List&lt;V1Toleration&gt;**](V1Toleration.md) |  |  [optional]
**ttlSecondsAfterFinished** | **Integer** | TTLSecondsAfterFinished limits the lifetime of a Workflow that has finished execution (Succeeded, Failed, Error). If this field is set, once the Workflow finishes, it will be deleted after ttlSecondsAfterFinished expires. If this field is unset, ttlSecondsAfterFinished will not expire. If this field is set to zero, ttlSecondsAfterFinished expires immediately after the Workflow finishes. DEPRECATED: Use TTLStrategy.SecondsAfterCompletion instead. |  [optional]
**ttlStrategy** | [**V1alpha1TTLStrategy**](V1alpha1TTLStrategy.md) |  |  [optional]
**volumeClaimTemplates** | [**List&lt;V1PersistentVolumeClaim&gt;**](V1PersistentVolumeClaim.md) |  |  [optional]
**volumes** | [**List&lt;V1Volume&gt;**](V1Volume.md) |  |  [optional]



