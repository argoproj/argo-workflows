

# Argo Fields


## Workflow

Workflow is the definition of a workflow resource

<details>
<summary>Examples (click to open)</summary>
<br>

- [`archive-location.yaml`](../examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`default-pdb-support.yaml`](../examples/default-pdb-support.yaml)

- [`dns-config.yaml`](../examples/dns-config.yaml)

- [`exit-handlers.yaml`](../examples/exit-handlers.yaml)

- [`forever.yaml`](../examples/forever.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](../examples/gc-ttl.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`hello-world.yaml`](../examples/hello-world.yaml)

- [`image-pull-secrets.yaml`](../examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`init-container.yaml`](../examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-owner-reference.yaml`](../examples/k8s-owner-reference.yaml)

- [`k8s-set-owner-reference.yaml`](../examples/k8s-set-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](../examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](../examples/retry-container.yaml)

- [`retry-on-error.yaml`](../examples/retry-on-error.yaml)

- [`retry-script.yaml`](../examples/retry-script.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](../examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](../examples/sidecar.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)

- [`template-on-exit.yaml`](../examples/template-on-exit.yaml)

- [`timeouts-step.yaml`](../examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](../examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`metadata`|[`ObjectMeta`](#objectmeta)|_No description available_|
|`spec`|[`WorkflowSpec`](#workflowspec)|_No description available_|
|`status`|[`WorkflowStatus`](#workflowstatus)|_No description available_|

## CronWorkflow

CronWorkflow is the definition of a scheduled workflow resource

<details>
<summary>Examples (click to open)</summary>
<br>

- [`cron-workflow.yaml`](../examples/cron-workflow.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`metadata`|[`ObjectMeta`](#objectmeta)|_No description available_|
|`spec`|[`CronWorkflowSpec`](#cronworkflowspec)|_No description available_|
|`status`|[`CronWorkflowStatus`](#cronworkflowstatus)|_No description available_|

## WorkflowTemplate

WorkflowTemplate is the definition of a workflow template resource

<details>
<summary>Examples (click to open)</summary>
<br>

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`metadata`|[`ObjectMeta`](#objectmeta)|_No description available_|
|`spec`|[`WorkflowTemplateSpec`](#workflowtemplatespec)|_No description available_|

## WorkflowSpec

WorkflowSpec is the specification of a Workflow.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](../examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`cron-workflow.yaml`](../examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`default-pdb-support.yaml`](../examples/default-pdb-support.yaml)

- [`dns-config.yaml`](../examples/dns-config.yaml)

- [`exit-handlers.yaml`](../examples/exit-handlers.yaml)

- [`forever.yaml`](../examples/forever.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](../examples/gc-ttl.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`hello-world.yaml`](../examples/hello-world.yaml)

- [`image-pull-secrets.yaml`](../examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`init-container.yaml`](../examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-owner-reference.yaml`](../examples/k8s-owner-reference.yaml)

- [`k8s-set-owner-reference.yaml`](../examples/k8s-set-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](../examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](../examples/retry-container.yaml)

- [`retry-on-error.yaml`](../examples/retry-on-error.yaml)

- [`retry-script.yaml`](../examples/retry-script.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](../examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](../examples/sidecar.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)

- [`template-on-exit.yaml`](../examples/template-on-exit.yaml)

- [`testvolume.yaml`](../examples/testvolume.yaml)

- [`timeouts-step.yaml`](../examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](../examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`activeDeadlineSeconds`|`int64`|Optional duration in seconds relative to the workflow start time which the workflow isallowed to run before the controller terminates the io.argoproj.workflow.v1alpha1. A value of zero is used toterminate a Running workflow|
|`affinity`|[`Affinity`](#affinity)|Affinity sets the scheduling constraints for all pods in the io.argoproj.workflow.v1alpha1.Can be overridden by an affinity specified in the template|
|`arguments`|[`Arguments`](#arguments)|Arguments contain the parameters and artifacts sent to the workflow entrypointParameters are referencable globally using the 'workflow' variable prefix.e.g. {{io.argoproj.workflow.v1alpha1.parameters.myparam}}|
|`artifactRepositoryRef`|[`ArtifactRepositoryRef`](#artifactrepositoryref)|ArtifactRepositoryRef specifies the configMap name and key containing the artifact repository config.|
|`automountServiceAccountToken`|`boolean`|AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods.ServiceAccountName of ExecutorConfig must be specified if this value is false.|
|`dnsConfig`|[`PodDNSConfig`](#poddnsconfig)|PodDNSConfig defines the DNS parameters of a pod in addition tothose generated from DNSPolicy.|
|`dnsPolicy`|`string`|Set DNS policy for the pod.Defaults to "ClusterFirst".Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'.DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy.To have DNS options set along with hostNetwork, you have to specify DNS policyexplicitly to 'ClusterFirstWithHostNet'.|
|`entrypoint`|`string`|Entrypoint is a template reference to the starting point of the io.argoproj.workflow.v1alpha1.|
|`executor`|[`ExecutorConfig`](#executorconfig)|Executor holds configurations of executor containers of the io.argoproj.workflow.v1alpha1.|
|`hostAliases`|`Array<`[`HostAlias`](#hostalias)`>`||
|`hostNetwork`|`boolean`|Host networking requested for this workflow pod. Default to false.|
|`imagePullSecrets`|`Array<`[`LocalObjectReference`](#localobjectreference)`>`|ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any imagesin pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secretscan be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet.More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod|
|`metrics`|[`Metrics`](#metrics)|Metrics are a list of metrics emitted from this Workflow|
|`nodeSelector`|`Map< string , string >`|NodeSelector is a selector which will result in all pods of the workflowto be scheduled on the selected node(s). This is able to be overridden bya nodeSelector specified in the template.|
|`onExit`|`string`|OnExit is a template reference which is invoked at the end of theworkflow, irrespective of the success, failure, or error of theprimary io.argoproj.workflow.v1alpha1.|
|`parallelism`|`int64`|Parallelism limits the max total parallel pods that can execute at the same time in a workflow|
|`podDisruptionBudget`|[`PodDisruptionBudgetSpec`](#poddisruptionbudgetspec)|PodDisruptionBudget holds the number of concurrent disruptions that you allow for Workflow's Pods.Controller will automatically add the selector with workflow name, if selector is empty.Optional: Defaults to empty.|
|`podGC`|[`PodGC`](#podgc)|PodGC describes the strategy to use when to deleting completed pods|
|`podPriority`|`int32`|Priority to apply to workflow pods.|
|`podPriorityClassName`|`string`|PriorityClassName to apply to workflow pods.|
|`podSpecPatch`|`string`|PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization ofcontainer fields which are not strings (e.g. resource limits).|
|`priority`|`int32`|Priority is used if controller is configured to process limited number of workflows in parallel. Workflows with higher priority are processed first.|
|`schedulerName`|`string`|Set scheduler name for all pods.Will be overridden if container/script template's scheduler name is set.Default scheduler will be used if neither specified.|
|`securityContext`|[`PodSecurityContext`](#podsecuritycontext)|SecurityContext holds pod-level security attributes and common container settings.Optional: Defaults to empty.  See type description for default values of each field.|
|`serviceAccountName`|`string`|ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as.|
|`shutdown`|`string`|Shutdown will shutdown the workflow according to its ShutdownStrategy|
|`suspend`|`boolean`|Suspend will suspend the workflow and prevent execution of any future steps in the workflow|
|`templates`|`Array<`[`Template`](#template)`>`|Templates is a list of workflow templates used in a workflow|
|`tolerations`|`Array<`[`Toleration`](#toleration)`>`|Tolerations to apply to workflow pods.|
|~`ttlSecondsAfterFinished`~|~`int32`~|~TTLSecondsAfterFinished limits the lifetime of a Workflow that has finished execution(Succeeded, Failed, Error). If this field is set, once the Workflow finishes, it will bedeleted after ttlSecondsAfterFinished expires. If this field is unset,ttlSecondsAfterFinished will not expire. If this field is set to zero,ttlSecondsAfterFinished expires immediately after the Workflow finishes.~ DEPRECATED: Use TTLStrategy.SecondsAfterCompletion instead.|
|`ttlStrategy`|[`TTLStrategy`](#ttlstrategy)|TTLStrategy limits the lifetime of a Workflow that has finished execution depending on if itSucceeded or Failed. If this struct is set, once the Workflow finishes, it will bedeleted after the time to live expires. If this field is unset,the controller config map will hold the default values.|
|`volumeClaimTemplates`|`Array<`[`PersistentVolumeClaim`](#persistentvolumeclaim)`>`|VolumeClaimTemplates is a list of claims that containers are allowed to reference.The Workflow controller will create the claims at the beginning of the workflowand delete the claims upon completion of the workflow|
|`volumes`|`Array<`[`Volume`](#volume)`>`|Volumes is a list of volumes that can be mounted by containers in a io.argoproj.workflow.v1alpha1.|

## WorkflowStatus

WorkflowStatus contains overall status information about a workflow

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`compressedNodes`|`string`|Compressed and base64 decoded Nodes map|
|`conditions`|`Array<`[`WorkflowCondition`](#workflowcondition)`>`|Conditions is a list of conditions the Workflow may have|
|`finishedAt`|[`Time`](#time)|Time at which this workflow completed|
|`message`|`string`|A human readable message indicating details about why the workflow is in this condition.|
|`nodes`|[`NodeStatus`](#nodestatus)|Nodes is a mapping between a node ID and the node's status.|
|`offloadNodeStatusVersion`|`string`|Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty.This will actually be populated with a hash of the offloaded data.|
|`outputs`|[`Outputs`](#outputs)|Outputs captures output values and artifact locations produced by the workflow via global outputs|
|`persistentVolumeClaims`|`Array<`[`Volume`](#volume)`>`|PersistentVolumeClaims tracks all PVCs that were created as part of the io.argoproj.workflow.v1alpha1.The contents of this list are drained at the end of the workflow.|
|`phase`|`string`|Phase a simple, high-level summary of where the workflow is in its lifecycle.|
|`resourcesDuration`|`Map< string , int64 >`|ResourcesDuration is the total for the workflow|
|`startedAt`|[`Time`](#time)|Time at which this workflow started|
|`storedTemplates`|[`Template`](#template)|StoredTemplates is a mapping between a template ref and the node's status.|

## CronWorkflowSpec

CronWorkflowSpec is the specification of a CronWorkflow

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](../examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`cron-workflow.yaml`](../examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`default-pdb-support.yaml`](../examples/default-pdb-support.yaml)

- [`dns-config.yaml`](../examples/dns-config.yaml)

- [`exit-handlers.yaml`](../examples/exit-handlers.yaml)

- [`forever.yaml`](../examples/forever.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](../examples/gc-ttl.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`hello-world.yaml`](../examples/hello-world.yaml)

- [`image-pull-secrets.yaml`](../examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`init-container.yaml`](../examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-owner-reference.yaml`](../examples/k8s-owner-reference.yaml)

- [`k8s-set-owner-reference.yaml`](../examples/k8s-set-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](../examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](../examples/retry-container.yaml)

- [`retry-on-error.yaml`](../examples/retry-on-error.yaml)

- [`retry-script.yaml`](../examples/retry-script.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](../examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](../examples/sidecar.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)

- [`template-on-exit.yaml`](../examples/template-on-exit.yaml)

- [`testvolume.yaml`](../examples/testvolume.yaml)

- [`timeouts-step.yaml`](../examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](../examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`concurrencyPolicy`|`string`|ConcurrencyPolicy is the K8s-style concurrency policy that will be used|
|`failedJobsHistoryLimit`|`int32`|FailedJobsHistoryLimit is the number of successful jobs to be kept at a time|
|`schedule`|`string`|Schedule is a schedule to run the Workflow in Cron format|
|`startingDeadlineSeconds`|`int64`|StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after itsoriginal scheduled time if it is missed.|
|`successfulJobsHistoryLimit`|`int32`|SuccessfulJobsHistoryLimit is the number of successful jobs to be kept at a time|
|`suspend`|`boolean`|Suspend is a flag that will stop new CronWorkflows from running if set to true|
|`timezone`|`string`|Timezone is the timezone against which the cron schedule will be calculated, e.g. "Asia/Tokyo". Default is machine's local time.|
|`workflowMeta`|[`ObjectMeta`](#objectmeta)|WorkflowMetadata contains some metadata of the workflow to be run|
|`workflowSpec`|[`WorkflowSpec`](#workflowspec)|WorkflowSpec is the spec of the workflow to be run|

## CronWorkflowStatus

CronWorkflowStatus is the status of a CronWorkflow

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`active`|`Array<`[`ObjectReference`](#objectreference)`>`|Active is a list of active workflows stemming from this CronWorkflow|
|`lastScheduledTime`|[`Time`](#time)|LastScheduleTime is the last time the CronWorkflow was scheduled|

## WorkflowTemplateSpec

WorkflowTemplateSpec is a spec of WorkflowTemplate.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](../examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`cron-workflow.yaml`](../examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`default-pdb-support.yaml`](../examples/default-pdb-support.yaml)

- [`dns-config.yaml`](../examples/dns-config.yaml)

- [`exit-handlers.yaml`](../examples/exit-handlers.yaml)

- [`forever.yaml`](../examples/forever.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](../examples/gc-ttl.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`hello-world.yaml`](../examples/hello-world.yaml)

- [`image-pull-secrets.yaml`](../examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`init-container.yaml`](../examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-owner-reference.yaml`](../examples/k8s-owner-reference.yaml)

- [`k8s-set-owner-reference.yaml`](../examples/k8s-set-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](../examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](../examples/retry-container.yaml)

- [`retry-on-error.yaml`](../examples/retry-on-error.yaml)

- [`retry-script.yaml`](../examples/retry-script.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](../examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](../examples/sidecar.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)

- [`template-on-exit.yaml`](../examples/template-on-exit.yaml)

- [`testvolume.yaml`](../examples/testvolume.yaml)

- [`timeouts-step.yaml`](../examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](../examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`workflowSpec`|[`WorkflowSpec`](#workflowspec)|_No description available_|

## Arguments

Arguments to a template

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifacts`|`Array<`[`Artifact`](#artifact)`>`|Artifacts is the list of artifacts to pass to the template or workflow|
|`parameters`|`Array<`[`Parameter`](#parameter)`>`|Parameters is the list of parameters to pass to the template or workflow|

## ArtifactRepositoryRef

_No description available_

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMap`|`string`|_No description available_|
|`key`|`string`|_No description available_|

## ExecutorConfig

ExecutorConfig holds configurations of an executor container.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`serviceAccountName`|`string`|ServiceAccountName specifies the service account name of the executor container.|

## Metrics

Metrics are a list of metrics emitted from a Workflow/Template

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`prometheus`|`Array<`[`Prometheus`](#prometheus)`>`|Prometheus is a list of prometheus metrics to be emitted|

## PodGC

PodGC describes how to delete completed pods as they complete

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`strategy`|`string`|Strategy is the strategy to use. One of "OnPodCompletion", "OnPodSuccess", "OnWorkflowCompletion", "OnWorkflowSuccess"|

## Template

Template is a reusable and composable unit of execution in a workflow

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](../examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`cron-workflow.yaml`](../examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`default-pdb-support.yaml`](../examples/default-pdb-support.yaml)

- [`dns-config.yaml`](../examples/dns-config.yaml)

- [`exit-handlers.yaml`](../examples/exit-handlers.yaml)

- [`forever.yaml`](../examples/forever.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](../examples/gc-ttl.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`hello-world.yaml`](../examples/hello-world.yaml)

- [`image-pull-secrets.yaml`](../examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`init-container.yaml`](../examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-owner-reference.yaml`](../examples/k8s-owner-reference.yaml)

- [`k8s-set-owner-reference.yaml`](../examples/k8s-set-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](../examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](../examples/retry-container.yaml)

- [`retry-on-error.yaml`](../examples/retry-on-error.yaml)

- [`retry-script.yaml`](../examples/retry-script.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](../examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](../examples/sidecar.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)

- [`template-on-exit.yaml`](../examples/template-on-exit.yaml)

- [`timeouts-step.yaml`](../examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](../examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`activeDeadlineSeconds`|`int64`|Optional duration in seconds relative to the StartTime that the pod may be active on a nodebefore the system actively tries to terminate the pod; value must be positive integerThis field is only applicable to container and script templates.|
|`affinity`|[`Affinity`](#affinity)|Affinity sets the pod's scheduling constraintsOverrides the affinity set at the workflow level (if any)|
|`archiveLocation`|[`ArtifactLocation`](#artifactlocation)|Location in which all files related to the step will be stored (logs, artifacts, etc...).Can be overridden by individual items in Outputs. If omitted, will use the defaultartifact repository location configured in the controller, appended with the<workflowname>/<nodename> in the key.|
|~`arguments`~|~[`Arguments`](#arguments)~|~Arguments hold arguments to the template.~ DEPRECATED: This field is not used.|
|`automountServiceAccountToken`|`boolean`|AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods.ServiceAccountName of ExecutorConfig must be specified if this value is false.|
|`container`|[`Container`](#container)|Container is the main container image to run in the pod|
|`daemon`|`boolean`|Deamon will allow a workflow to proceed to the next step so long as the container reaches readiness|
|`dag`|[`DAGTemplate`](#dagtemplate)|DAG template subtype which runs a DAG|
|`executor`|[`ExecutorConfig`](#executorconfig)|Executor holds configurations of the executor container.|
|`hostAliases`|`Array<`[`HostAlias`](#hostalias)`>`|HostAliases is an optional list of hosts and IPs that will be injected into the pod spec|
|`initContainers`|`Array<`[`UserContainer`](#usercontainer)`>`|InitContainers is a list of containers which run before the main container.|
|`inputs`|[`Inputs`](#inputs)|Inputs describe what inputs parameters and artifacts are supplied to this template|
|`metadata`|[`Metadata`](#metadata)|Metdata sets the pods's metadata, i.e. annotations and labels|
|`metrics`|[`Metrics`](#metrics)|Metrics are a list of metrics emitted from this template|
|`name`|`string`|Name is the name of the template|
|`nodeSelector`|`Map< string , string >`|NodeSelector is a selector to schedule this step of the workflow to berun on the selected node(s). Overrides the selector set at the workflow level.|
|`outputs`|[`Outputs`](#outputs)|Outputs describe the parameters and artifacts that this template produces|
|`parallelism`|`int64`|Parallelism limits the max total parallel pods that can execute at the same time within theboundaries of this template invocation. If additional steps/dag templates are invoked, thepods created by those templates will not be counted towards this total.|
|`podSpecPatch`|`string`|PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization ofcontainer fields which are not strings (e.g. resource limits).|
|`priority`|`int32`|Priority to apply to workflow pods.|
|`priorityClassName`|`string`|PriorityClassName to apply to workflow pods.|
|`resource`|[`ResourceTemplate`](#resourcetemplate)|Resource template subtype which can run k8s resources|
|`resubmitPendingPods`|`boolean`|ResubmitPendingPods is a flag to enable resubmitting pods that remain Pending after initial submission|
|`retryStrategy`|[`RetryStrategy`](#retrystrategy)|RetryStrategy describes how to retry a template when it fails|
|`schedulerName`|`string`|If specified, the pod will be dispatched by specified scheduler.Or it will be dispatched by workflow scope scheduler if specified.If neither specified, the pod will be dispatched by default scheduler.|
|`script`|[`ScriptTemplate`](#scripttemplate)|Script runs a portion of code against an interpreter|
|`securityContext`|[`PodSecurityContext`](#podsecuritycontext)|SecurityContext holds pod-level security attributes and common container settings.Optional: Defaults to empty.  See type description for default values of each field.|
|`serviceAccountName`|`string`|ServiceAccountName to apply to workflow pods|
|`sidecars`|`Array<`[`UserContainer`](#usercontainer)`>`|Sidecars is a list of containers which run alongside the main containerSidecars are automatically killed when the main container completes|
|`steps`|`Array<`[`ParallelSteps`](#parallelsteps)`>`|Steps define a series of sequential/parallel workflow steps|
|`suspend`|[`SuspendTemplate`](#suspendtemplate)|Suspend template subtype which can suspend a workflow when reaching the step|
|~`template`~|~`string`~|~Template is the name of the template which is used as the base of this template.~ DEPRECATED: This field is not used.|
|~`templateRef`~|~[`TemplateRef`](#templateref)~|~TemplateRef is the reference to the template resource which is used as the base of this template.~ DEPRECATED: This field is not used.|
|`tolerations`|`Array<`[`Toleration`](#toleration)`>`|Tolerations to apply to workflow pods.|
|`volumes`|`Array<`[`Volume`](#volume)`>`|Volumes is a list of volumes that can be mounted by containers in a template.|

## TTLStrategy

TTLStrategy is the strategy for the time to live depending on if the workflow succeeded or failed

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`gc-ttl.yaml`](../examples/gc-ttl.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`secondsAfterCompletion`|`int32`|SecondsAfterCompletion is the number of seconds to live after completion|
|`secondsAfterFailure`|`int32`|SecondsAfterFailure is the number of seconds to live after failure|
|`secondsAfterSuccess`|`int32`|SecondsAfterSuccess is the number of seconds to live after success|

## WorkflowCondition

_No description available_

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`message`|`string`|Message is the condition message|
|`status`|`string`|Status is the status of the condition|
|`type`|`string`|Type is the type of condition|

## NodeStatus

NodeStatus contains status information about an individual node in the workflow

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`boundaryID`|`string`|BoundaryID indicates the node ID of the associated template root node in which this node belongs to|
|`children`|`Array< string >`|Children is a list of child node IDs|
|`daemoned`|`boolean`|Daemoned tracks whether or not this node was daemoned and need to be terminated|
|`displayName`|`string`|DisplayName is a human readable representation of the node. Unique within a template boundary|
|`finishedAt`|[`Time`](#time)|Time at which this node completed|
|`id`|`string`|ID is a unique identifier of a node within the worklowIt is implemented as a hash of the node name, which makes the ID deterministic|
|`inputs`|[`Inputs`](#inputs)|Inputs captures input parameter values and artifact locations supplied to this template invocation|
|`message`|`string`|A human readable message indicating details about why the node is in this condition.|
|`name`|`string`|Name is unique name in the node tree used to generate the node ID|
|`outboundNodes`|`Array< string >`|OutboundNodes tracks the node IDs which are considered "outbound" nodes to a template invocation.For every invocation of a template, there are nodes which we considered as "outbound". Essentially,these are last nodes in the execution sequence to run, before the template is considered completed.These nodes are then connected as parents to a following step.In the case of single pod steps (i.e. container, script, resource templates), this list will be nilsince the pod itself is already considered the "outbound" node.In the case of DAGs, outbound nodes are the "target" tasks (tasks with no children).In the case of steps, outbound nodes are all the containers involved in the last step group.NOTE: since templates are composable, the list of outbound nodes are carried upwards whena DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes ofa template, will be a superset of the outbound nodes of its last children.|
|`outputs`|[`Outputs`](#outputs)|Outputs captures output parameter values and artifact locations produced by this template invocation|
|`phase`|`string`|Phase a simple, high-level summary of where the node is in its lifecycle.Can be used as a state machine.|
|`podIP`|`string`|PodIP captures the IP of the pod for daemoned steps|
|`resourcesDuration`|`Map< string , int64 >`|ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes.|
|`startedAt`|[`Time`](#time)|Time at which this node started|
|~`storedTemplateID`~|~`string`~|~StoredTemplateID is the ID of stored template.~ DEPRECATED: This value is not used anymore.|
|`templateName`|`string`|TemplateName is the template name which this node corresponds to.Not applicable to virtual nodes (e.g. Retry, StepGroup)|
|`templateRef`|[`TemplateRef`](#templateref)|TemplateRef is the reference to the template resource which this node corresponds to.Not applicable to virtual nodes (e.g. Retry, StepGroup)|
|`templateScope`|`string`|TemplateScope is the template scope in which the template of this node was retrieved.|
|`type`|`string`|Type indicates type of node|
|~`workflowTemplateName`~|~`string`~|~WorkflowTemplateName is the WorkflowTemplate resource name on which the resolved template of this node is retrieved.~ DEPRECATED: This value is not used anymore.|

## Outputs

Outputs hold parameters, artifacts, and results from a step

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifacts`|`Array<`[`Artifact`](#artifact)`>`|Artifacts holds the list of output artifacts produced by a step|
|`parameters`|`Array<`[`Parameter`](#parameter)`>`|Parameters holds the list of output parameters produced by a step|
|`result`|`string`|Result holds the result (stdout) of a script template|

## Artifact

Artifact indicates an artifact to place at a specified path

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`archive`|[`ArchiveStrategy`](#archivestrategy)|Archive controls how the artifact will be saved to the artifact repository.|
|`artifactLocation`|[`ArtifactLocation`](#artifactlocation)|ArtifactLocation contains the location of the artifact|
|`from`|`string`|From allows an artifact to reference an artifact from a previous step|
|`globalName`|`string`|GlobalName exports an output artifact to the global scope, making it available as'{{io.argoproj.workflow.v1alpha1.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts|
|`mode`|`int32`|mode bits to use on this file, must be a value between 0 and 0777set when loading input artifacts.|
|`name`|`string`|name of the artifact. must be unique within a template's inputs/outputs.|
|`optional`|`boolean`|Make Artifacts optional, if Artifacts doesn't generate or exist|
|`path`|`string`|Path is the container path to the artifact|

## Parameter

Parameter indicate a passed string parameter to a service template with an optional default value

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|~`default`~|~`string`~|~Default is the default value to use for an input parameter if a value was not supplied~ DEPRECATED: This field is not used|
|`globalName`|`string`|GlobalName exports an output parameter to the global scope, making it available as'{{io.argoproj.workflow.v1alpha1.outputs.parameters.XXXX}} and in workflow.status.outputs.parameters|
|`name`|`string`|Name is the parameter name|
|`value`|`string`|Value is the literal value to use for the parameter.If specified in the context of an input parameter, the value takes precedence over any passed values|
|`valueFrom`|[`ValueFrom`](#valuefrom)|ValueFrom is the source for the output parameter's value|

## Prometheus

Prometheus is a prometheus metric to be emitted

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`counter`|[`Counter`](#counter)|Counter is a counter metric|
|`gauge`|[`Gauge`](#gauge)|Gauge is a gauge metric|
|`help`|`string`|Help is a string that describes the metric|
|`histogram`|[`Histogram`](#histogram)|Histogram is a histogram metric|
|`labels`|`Array<`[`MetricLabel`](#metriclabel)`>`|Labels is a list of metric labels|
|`name`|`string`|Name is the name of the metric|
|`when`|`string`|When is a conditional statement that decides when to emit the metric|

## ArtifactLocation

ArtifactLocation describes a location for a single or multiple artifacts.It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname).It is also used to describe the location of multiple artifacts such as the archive locationof a single workflow step, which the executor will use as a default location to store its files.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](../examples/archive-location.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`archiveLogs`|`boolean`|ArchiveLogs indicates if the container logs should be archived|
|`artifactory`|[`ArtifactoryArtifact`](#artifactoryartifact)|Artifactory contains artifactory artifact location details|
|`gcs`|[`GCSArtifact`](#gcsartifact)|GCS contains GCS artifact location details|
|`git`|[`GitArtifact`](#gitartifact)|Git contains git artifact location details|
|`hdfs`|[`HDFSArtifact`](#hdfsartifact)|HDFS contains HDFS artifact location details|
|`http`|[`HTTPArtifact`](#httpartifact)|HTTP contains HTTP artifact location details|
|`oss`|[`OSSArtifact`](#ossartifact)|OSS contains OSS artifact location details|
|`raw`|[`RawArtifact`](#rawartifact)|Raw contains raw artifact location details|
|`s3`|[`S3Artifact`](#s3artifact)|S3 contains S3 artifact location details|

## DAGTemplate

DAGTemplate is a template subtype for directed acyclic graph templates

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`failFast`|`boolean`|This flag is for DAG logic. The DAG logic has a built-in "fail fast" feature to stop scheduling new steps,as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completedbefore failing the DAG itself.The FailFast flag default is true,  if set to false, it will allow a DAG to run all branches of the DAG tocompletion (either success or failure), regardless of the failed outcomes of branches in the DAG.More info and example about this feature at https://github.com/argoproj/argo/issues/1442|
|`target`|`string`|Target are one or more names of targets to execute in a DAG|
|`tasks`|`Array<`[`DAGTask`](#dagtask)`>`|Tasks are a list of DAG tasks|

## UserContainer

UserContainer is a container specified by a user.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`init-container.yaml`](../examples/init-container.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`container`|[`Container`](#container)|_No description available_|
|`mirrorVolumeMounts`|`boolean`|MirrorVolumeMounts will mount the same volumes specified in the main containerto the container (including artifacts), at the same mountPaths. This enablesdind daemon to partially see the same filesystem as the main container inorder to use features such as docker volume binding|

## Inputs

Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifacts`|`Array<`[`Artifact`](#artifact)`>`|Artifact are a list of artifacts passed as inputs|
|`parameters`|`Array<`[`Parameter`](#parameter)`>`|Parameters are a list of parameters passed as inputs|

## Metadata

Pod metdata

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](../examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`cron-workflow.yaml`](../examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`default-pdb-support.yaml`](../examples/default-pdb-support.yaml)

- [`dns-config.yaml`](../examples/dns-config.yaml)

- [`exit-handlers.yaml`](../examples/exit-handlers.yaml)

- [`forever.yaml`](../examples/forever.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](../examples/gc-ttl.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`hello-world.yaml`](../examples/hello-world.yaml)

- [`image-pull-secrets.yaml`](../examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`init-container.yaml`](../examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-owner-reference.yaml`](../examples/k8s-owner-reference.yaml)

- [`k8s-set-owner-reference.yaml`](../examples/k8s-set-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](../examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](../examples/retry-container.yaml)

- [`retry-on-error.yaml`](../examples/retry-on-error.yaml)

- [`retry-script.yaml`](../examples/retry-script.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](../examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](../examples/sidecar.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)

- [`template-on-exit.yaml`](../examples/template-on-exit.yaml)

- [`testvolume.yaml`](../examples/testvolume.yaml)

- [`timeouts-step.yaml`](../examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](../examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`annotations`|`Map< string , string >`|_No description available_|
|`labels`|`Map< string , string >`|_No description available_|

## ResourceTemplate

ResourceTemplate is a template subtype to manipulate kubernetes resources

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-owner-reference.yaml`](../examples/k8s-owner-reference.yaml)

- [`k8s-set-owner-reference.yaml`](../examples/k8s-set-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`action`|`string`|Action is the action to perform to the resource.Must be one of: get, create, apply, delete, replace, patch|
|`failureCondition`|`string`|FailureCondition is a label selector expression which describes the conditionsof the k8s resource in which the step was considered failed|
|`manifest`|`string`|Manifest contains the kubernetes manifest|
|`mergeStrategy`|`string`|MergeStrategy is the strategy used to merge a patch. It defaults to "strategic"Must be one of: strategic, merge, json|
|`setOwnerReference`|`boolean`|SetOwnerReference sets the reference to the workflow on the OwnerReference of generated resource.|
|`successCondition`|`string`|SuccessCondition is a label selector expression which describes the conditionsof the k8s resource in which it is acceptable to proceed to the following step|

## RetryStrategy

RetryStrategy provides controls on how to retry a workflow step

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](../examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](../examples/retry-container.yaml)

- [`retry-on-error.yaml`](../examples/retry-on-error.yaml)

- [`retry-script.yaml`](../examples/retry-script.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`backoff`|[`Backoff`](#backoff)|Backoff is a backoff strategy|
|`limit`|`int32`|Limit is the maximum number of attempts when retrying a container|
|`retryPolicy`|`string`|RetryPolicy is a policy of NodePhase statuses that will be retried|

## ScriptTemplate

ScriptTemplate is a template subtype to enable scripting through code steps

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`retry-script.yaml`](../examples/retry-script.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`container`|[`Container`](#container)|_No description available_|
|`source`|`string`|Source contains the source code of the script to execute|

## WorkflowStep

WorkflowStep is a reference to a template to execute in a series of step

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`exit-handlers.yaml`](../examples/exit-handlers.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)

- [`template-on-exit.yaml`](../examples/template-on-exit.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](../examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`arguments`|[`Arguments`](#arguments)|Arguments hold arguments to the template|
|`continueOn`|[`ContinueOn`](#continueon)|ContinueOn makes argo to proceed with the following step even if this step fails.Errors and Failed states can be specified|
|`name`|`string`|Name of the step|
|`onExit`|`string`|OnExit is a template reference which is invoked at the end of thetemplate, irrespective of the success, failure, or error of theprimary template.|
|`template`|`string`|Template is the name of the template to execute as the step|
|`templateRef`|[`TemplateRef`](#templateref)|TemplateRef is the reference to the template resource to execute as the step.|
|`when`|`string`|When is an expression in which the step should conditionally execute|
|`withItems`|`Array<`[`Item`](#item)`>`|WithItems expands a step into multiple parallel steps from the items in the list|
|`withParam`|`string`|WithParam expands a step into multiple parallel steps from the value in the parameter,which is expected to be a JSON list.|
|`withSequence`|[`Sequence`](#sequence)|WithSequence expands a step into a numeric sequence|

## SuspendTemplate

SuspendTemplate is a template subtype to suspend a workflow at a predetermined point in time

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`cron-workflow.yaml`](../examples/cron-workflow.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`duration`|`string`|Duration is the seconds to wait before automatically resuming a template|

## TemplateRef

TemplateRef is a reference of template resource.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](../examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`clusterscope`|`boolean`|ClusterScope indicates the referred template is cluster scoped (i.e., a ClusterWorkflowTemplate).|
|`name`|`string`|Name is the resource name of the template.|
|`runtimeResolution`|`boolean`|RuntimeResolution skips validation at creation time.By enabling this option, you can create the referred workflow template before the actual runtime.|
|`template`|`string`|Template is the name of referred template in the resource.|

## ArchiveStrategy

ArchiveStrategy describes how to archive files/directory when saving artifacts

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`none`|[`NoneStrategy`](#nonestrategy)|_No description available_|
|`tar`|[`TarStrategy`](#tarstrategy)|_No description available_|

## ValueFrom

ValueFrom describes a location in which to obtain the value to a parameter

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`default`|`string`|Default specifies a value to be used if retrieving the value from the specified source fails|
|`jqFilter`|`string`|JQFilter expression against the resource object in resource templates|
|`jsonPath`|`string`|JSONPath of a resource to retrieve an output parameter value from in resource templates|
|`parameter`|`string`|Parameter reference to a step or dag task in which to retrieve an output parameter value from(e.g. '{{steps.mystep.outputs.myparam}}')|
|`path`|`string`|Path in the container to retrieve an output parameter value from in container templates|

## Counter

Counter is a Counter prometheus metric

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`value`|`string`|Value is the value of the metric|

## Gauge

Gauge is a Gauge prometheus metric

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`realtime`|`boolean`|Realtime emits this metric in real time if applicable|
|`value`|`string`|Value is the value of the metric|

## Histogram

Histogram is a Histogram prometheus metric

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`buckets`|`Array< number >`|Buckets is a list of bucket divisors for the histogram|
|`value`|`string`|Value is the value of the metric|

## MetricLabel

MetricLabel is a single label for a prometheus metric

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`forever.yaml`](../examples/forever.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|_No description available_|
|`value`|`string`|_No description available_|

## ArtifactoryArtifact

ArtifactoryArtifact is the location of an artifactory artifact

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifactoryAuth`|[`ArtifactoryAuth`](#artifactoryauth)|_No description available_|
|`url`|`string`|URL of the artifact|

## GCSArtifact

GCSArtifact is the location of a GCS artifact

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`gCSBucket`|[`GCSBucket`](#gcsbucket)|_No description available_|
|`key`|`string`|Key is the path in the bucket where the artifact resides|

## GitArtifact

GitArtifact is the location of an git artifact

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`depth`|`uint64`|Depth specifies clones/fetches should be shallow and include the givennumber of commits from the branch tip|
|`fetch`|`Array< string >`|Fetch specifies a number of refs that should be fetched before checkout|
|`insecureIgnoreHostKey`|`boolean`|InsecureIgnoreHostKey disables SSH strict host key checking during git clone|
|`passwordSecret`|[`SecretKeySelector`](#secretkeyselector)|PasswordSecret is the secret selector to the repository password|
|`repo`|`string`|Repo is the git repository|
|`revision`|`string`|Revision is the git commit, tag, branch to checkout|
|`sshPrivateKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SSHPrivateKeySecret is the secret selector to the repository ssh private key|
|`usernameSecret`|[`SecretKeySelector`](#secretkeyselector)|UsernameSecret is the secret selector to the repository username|

## HDFSArtifact

HDFSArtifact is the location of an HDFS artifact

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`force`|`boolean`|Force copies a file forcibly even if it exists (default: false)|
|`hDFSConfig`|[`HDFSConfig`](#hdfsconfig)|_No description available_|
|`path`|`string`|Path is a file path in HDFS|

## HTTPArtifact

HTTPArtifact allows an file served on HTTP to be placed as an input artifact in a container

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`sidecar-nginx.yaml`](../examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](../examples/sidecar.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`url`|`string`|URL of the artifact|

## OSSArtifact

OSSArtifact is the location of an OSS artifact

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|Key is the path in the bucket where the artifact resides|
|`oSSBucket`|[`OSSBucket`](#ossbucket)|_No description available_|

## RawArtifact

RawArtifact allows raw string content to be placed as an artifact in a container

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`data`|`string`|Data is the string contents of the artifact|

## S3Artifact

S3Artifact is the location of an S3 artifact

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|Key is the key in the bucket where the artifact resides|
|`s3Bucket`|[`S3Bucket`](#s3bucket)|_No description available_|

## DAGTask

DAGTask represents a node in the graph during DAG execution

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`arguments`|[`Arguments`](#arguments)|Arguments are the parameter and artifact arguments to the template|
|`continueOn`|[`ContinueOn`](#continueon)|ContinueOn makes argo to proceed with the following step even if this step fails.Errors and Failed states can be specified|
|`dependencies`|`Array< string >`|Dependencies are name of other targets which this depends on|
|`name`|`string`|Name is the name of the target|
|`onExit`|`string`|OnExit is a template reference which is invoked at the end of thetemplate, irrespective of the success, failure, or error of theprimary template.|
|`template`|`string`|Name of template to execute|
|`templateRef`|[`TemplateRef`](#templateref)|TemplateRef is the reference to the template resource to execute.|
|`when`|`string`|When is an expression in which the task should conditionally execute|
|`withItems`|`Array<`[`Item`](#item)`>`|WithItems expands a task into multiple parallel tasks from the items in the list|
|`withParam`|`string`|WithParam expands a task into multiple parallel tasks from the value in the parameter,which is expected to be a JSON list.|
|`withSequence`|[`Sequence`](#sequence)|WithSequence expands a task into a numeric sequence|

## Backoff

Backoff is a backoff strategy to use within retryStrategy

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`duration`|`string`|Duration is the amount to back off. Default unit is seconds, but could also be a duration (e.g. "2m", "1h")|
|`factor`|`int32`|Factor is a factor to multiply the base duration after each failed retry|
|`maxDuration`|`string`|MaxDuration is the maximum amount of time allowed for the backoff strategy|

## ContinueOn

ContinueOn defines if a workflow should continue even if a task or step fails/errors.It can be specified if the workflow should continue when the pod errors, fails or both.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`error`|`boolean`||
|`failed`|`boolean`||

## Item



<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`boolVal`|`boolean`|_No description available_|
|`listVal`|`Array<`[`ItemValue`](#itemvalue)`>`|_No description available_|
|`mapVal`|[`ItemValue`](#itemvalue)|_No description available_|
|`numVal`|`string`|_No description available_|
|`strVal`|`string`|_No description available_|
|`type`|`int64`|_No description available_|

## Sequence

Sequence expands a workflow step into numeric range

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`count`|`string`|Count is number of elements in the sequence (default: 0). Not to be used with end|
|`end`|`string`|Number at which to end the sequence (default: 0). Not to be used with Count|
|`format`|`string`|Format is a printf format string to format the value in the sequence|
|`start`|`string`|Number at which to start the sequence (default: 0)|

## NoneStrategy

NoneStrategy indicates to skip tar process and upload the files or directory tree as independentfiles. Note that if the artifact is a directory, the artifact driver must support the ability tosave/load the directory appropriately.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)
</details>

## TarStrategy

TarStrategy will tar and gzip the file or directory when saving

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`compressionLevel`|`int32`|CompressionLevel specifies the gzip compression level to use for the artifact.Defaults to gzip.DefaultCompression.|

## ArtifactoryAuth

ArtifactoryAuth describes the secret selectors required for authenticating to artifactory

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`passwordSecret`|[`SecretKeySelector`](#secretkeyselector)|PasswordSecret is the secret selector to the repository password|
|`usernameSecret`|[`SecretKeySelector`](#secretkeyselector)|UsernameSecret is the secret selector to the repository username|

## GCSBucket

GCSBucket contains the access information for interfacring with a GCS bucket

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`bucket`|`string`|Bucket is the name of the bucket|
|`serviceAccountKeySecret`|[`SecretKeySelector`](#secretkeyselector)|ServiceAccountKeySecret is the secret selector to the bucket's service account key|

## HDFSConfig

HDFSConfig is configurations for HDFS

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`addresses`|`Array< string >`|Addresses is accessible addresses of HDFS name nodes|
|`hDFSKrbConfig`|[`HDFSKrbConfig`](#hdfskrbconfig)|_No description available_|
|`hdfsUser`|`string`|HDFSUser is the user to access HDFS file system.It is ignored if either ccache or keytab is used.|

## OSSBucket

OSSBucket contains the access information required for interfacing with an OSS bucket

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessKeySecret`|[`SecretKeySelector`](#secretkeyselector)|AccessKeySecret is the secret selector to the bucket's access key|
|`bucket`|`string`|Bucket is the name of the bucket|
|`endpoint`|`string`|Endpoint is the hostname of the bucket endpoint|
|`secretKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySecret is the secret selector to the bucket's secret key|

## S3Bucket

S3Bucket contains the access information required for interfacing with an S3 bucket

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessKeySecret`|[`SecretKeySelector`](#secretkeyselector)|AccessKeySecret is the secret selector to the bucket's access key|
|`bucket`|`string`|Bucket is the name of the bucket|
|`endpoint`|`string`|Endpoint is the hostname of the bucket endpoint|
|`insecure`|`boolean`|Insecure will connect to the service with TLS|
|`region`|`string`|Region contains the optional bucket region|
|`roleARN`|`string`|RoleARN is the Amazon Resource Name (ARN) of the role to assume.|
|`secretKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySecret is the secret selector to the bucket's secret key|
|`useSDKCreds`|`boolean`|UseSDKCreds tells the driver to figure out credentials based on sdk defaults.|

## ItemValue



### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`boolVal`|`boolean`|_No description available_|
|`listVal`|`Array< string >`|_No description available_|
|`mapVal`|`Map< string , string >`|_No description available_|
|`numVal`|`string`|_No description available_|
|`strVal`|`string`|_No description available_|
|`type`|`int64`|_No description available_|

## HDFSKrbConfig

HDFSKrbConfig is auth configurations for Kerberos

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`krbCCacheSecret`|[`SecretKeySelector`](#secretkeyselector)|KrbCCacheSecret is the secret selector for Kerberos ccacheEither ccache or keytab can be set to use Kerberos.|
|`krbConfigConfigMap`|[`ConfigMapKeySelector`](#configmapkeyselector)|KrbConfig is the configmap selector for Kerberos config as stringIt must be set if either ccache or keytab is used.|
|`krbKeytabSecret`|[`SecretKeySelector`](#secretkeyselector)|KrbKeytabSecret is the secret selector for Kerberos keytabEither ccache or keytab can be set to use Kerberos.|
|`krbRealm`|`string`|KrbRealm is the Kerberos realm used with Kerberos keytabIt must be set if keytab is used.|
|`krbServicePrincipalName`|`string`|KrbServicePrincipalName is the principal name of Kerberos serviceIt must be set if either ccache or keytab is used.|
|`krbUsername`|`string`|KrbUsername is the Kerberos username used with Kerberos keytabIt must be set if keytab is used.|

# External Fields


## ObjectMeta

ObjectMeta is metadata that all persisted resources must have, which includes all objectsusers must create.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](../examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`cron-workflow.yaml`](../examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`default-pdb-support.yaml`](../examples/default-pdb-support.yaml)

- [`dns-config.yaml`](../examples/dns-config.yaml)

- [`exit-handlers.yaml`](../examples/exit-handlers.yaml)

- [`forever.yaml`](../examples/forever.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](../examples/gc-ttl.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`hello-world.yaml`](../examples/hello-world.yaml)

- [`image-pull-secrets.yaml`](../examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`init-container.yaml`](../examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-owner-reference.yaml`](../examples/k8s-owner-reference.yaml)

- [`k8s-set-owner-reference.yaml`](../examples/k8s-set-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](../examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](../examples/retry-container.yaml)

- [`retry-on-error.yaml`](../examples/retry-on-error.yaml)

- [`retry-script.yaml`](../examples/retry-script.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](../examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](../examples/sidecar.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)

- [`template-on-exit.yaml`](../examples/template-on-exit.yaml)

- [`testvolume.yaml`](../examples/testvolume.yaml)

- [`timeouts-step.yaml`](../examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](../examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`annotations`|`Map< string , string >`|Annotations is an unstructured key value map stored with a resource that may beset by external tools to store and retrieve arbitrary metadata. They are notqueryable and should be preserved when modifying objects.More info: http://kubernetes.io/docs/user-guide/annotations|
|`clusterName`|`string`|The name of the cluster which the object belongs to.This is used to distinguish resources with same name and namespace in different clusters.This field is not set anywhere right now and apiserver is going to ignore it if set in create or update request.|
|`creationTimestamp`|[`Time`](#time)|CreationTimestamp is a timestamp representing the server time when this object wascreated. It is not guaranteed to be set in happens-before order across separate operations.Clients may not set this value. It is represented in RFC3339 form and is in UTC.Populated by the system.Read-only.Null for lists.More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata|
|`deletionGracePeriodSeconds`|`int64`|Number of seconds allowed for this object to gracefully terminate beforeit will be removed from the system. Only set when deletionTimestamp is also set.May only be shortened.Read-only.|
|`deletionTimestamp`|[`Time`](#time)|DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. Thisfield is set by the server when a graceful deletion is requested by the user, and is notdirectly settable by a client. The resource is expected to be deleted (no longer visiblefrom resource lists, and not reachable by name) after the time in this field, once thefinalizers list is empty. As long as the finalizers list contains items, deletion is blocked.Once the deletionTimestamp is set, this value may not be unset or be set further into thefuture, although it may be shortened or the resource may be deleted prior to this time.For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will reactby sending a graceful termination signal to the containers in the pod. After that 30 seconds,the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup,remove the pod from the API. In the presence of network partitions, this object may stillexist after this timestamp, until an administrator or automated process can determine theresource is fully terminated.If not set, graceful deletion of the object has not been requested.Populated by the system when a graceful deletion is requested.Read-only.More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata|
|`finalizers`|`Array< string >`|Must be empty before the object is deleted from the registry. Each entryis an identifier for the responsible component that will remove the entryfrom the list. If the deletionTimestamp of the object is non-nil, entriesin this list can only be removed.|
|`generateName`|`string`|GenerateName is an optional prefix, used by the server, to generate a uniquename ONLY IF the Name field has not been provided.If this field is used, the name returned to the client will be differentthan the name passed. This value will also be combined with a unique suffix.The provided value has the same validation rules as the Name field,and may be truncated by the length of the suffix required to make the valueunique on the server.If this field is specified and the generated name exists, the server willNOT return a 409 - instead, it will either return 201 Created or 500 with ReasonServerTimeout indicating a unique name could not be found in the time allotted, and the clientshould retry (optionally after the time indicated in the Retry-After header).Applied only if Name is not specified.More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency|
|`generation`|`int64`|A sequence number representing a specific generation of the desired state.Populated by the system. Read-only.|
|`labels`|`Map< string , string >`|Map of string keys and values that can be used to organize and categorize(scope and select) objects. May match selectors of replication controllersand services.More info: http://kubernetes.io/docs/user-guide/labels|
|`managedFields`|`Array<`[`ManagedFieldsEntry`](#managedfieldsentry)`>`|ManagedFields maps workflow-id and version to the set of fieldsthat are managed by that io.argoproj.workflow.v1alpha1. This is mostly for internalhousekeeping, and users typically shouldn't need to set orunderstand this field. A workflow can be the user's name, acontroller's name, or the name of a specific apply path like"ci-cd". The set of fields is always in the version that theworkflow used when modifying the object.|
|`name`|`string`|Name must be unique within a namespace. Is required when creating resources, althoughsome resources may allow a client to request the generation of an appropriate nameautomatically. Name is primarily intended for creation idempotence and configurationdefinition.Cannot be updated.More info: http://kubernetes.io/docs/user-guide/identifiers#names|
|`namespace`|`string`|Namespace defines the space within each name must be unique. An empty namespace isequivalent to the "default" namespace, but "default" is the canonical representation.Not all objects are required to be scoped to a namespace - the value of this field forthose objects will be empty.Must be a DNS_LABEL.Cannot be updated.More info: http://kubernetes.io/docs/user-guide/namespaces|
|`ownerReferences`|`Array<`[`OwnerReference`](#ownerreference)`>`|List of objects depended by this object. If ALL objects in the list havebeen deleted, this object will be garbage collected. If this object is managed by a controller,then an entry in this list will point to this controller, with the controller field set to true.There cannot be more than one managing controller.|
|`resourceVersion`|`string`|An opaque value that represents the internal version of this object that canbe used by clients to determine when objects have changed. May be used for optimisticconcurrency, change detection, and the watch operation on a resource or set of resources.Clients must treat these values as opaque and passed unmodified back to the server.They may only be valid for a particular resource or set of resources.Populated by the system.Read-only.Value must be treated as opaque by clients and .More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency|
|~`selfLink`~|~`string`~|~SelfLink is a URL representing this object.Populated by the system.Read-only.~ DEPRECATEDKubernetes will stop propagating this field in 1.20 release and the field is plannedto be removed in 1.21 release.|
|`uid`|`string`|UID is the unique in time and space value for this object. It is typically generated bythe server on successful creation of a resource and is not allowed to change on PUToperations.Populated by the system.Read-only.More info: http://kubernetes.io/docs/user-guide/identifiers#uids|

## Affinity

Affinity is a group of affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nodeAffinity`|[`NodeAffinity`](#nodeaffinity)|Describes node affinity scheduling rules for the pod.|
|`podAffinity`|[`PodAffinity`](#podaffinity)|Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).|
|`podAntiAffinity`|[`PodAntiAffinity`](#podantiaffinity)|Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).|

## PodDNSConfig

PodDNSConfig defines the DNS parameters of a pod in addition tothose generated from DNSPolicy.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`dns-config.yaml`](../examples/dns-config.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nameservers`|`Array< string >`|A list of DNS name server IP addresses.This will be appended to the base nameservers generated from DNSPolicy.Duplicated nameservers will be removed.|
|`options`|`Array<`[`PodDNSConfigOption`](#poddnsconfigoption)`>`|A list of DNS resolver options.This will be merged with the base options generated from DNSPolicy.Duplicated entries will be removed. Resolution options given in Optionswill override those that appear in the base DNSPolicy.|
|`searches`|`Array< string >`|A list of DNS search domains for host-name lookup.This will be appended to the base search paths generated from DNSPolicy.Duplicated search paths will be removed.|

## HostAlias

HostAlias holds the mapping between IP and hostnames that will be injected as an entry in thepod's hosts file.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`hostnames`|`Array< string >`|Hostnames for the above IP address.|
|`ip`|`string`|IP address of the host file entry.|

## LocalObjectReference

LocalObjectReference contains enough information to let you locate thereferenced object inside the same namespace.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`image-pull-secrets.yaml`](../examples/image-pull-secrets.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Name of the referent.More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#namesTODO: Add other useful fields. apiVersion, kind, uid?|

## PodDisruptionBudgetSpec

PodDisruptionBudgetSpec is a description of a PodDisruptionBudget.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`maxUnavailable`|[`IntOrString`](#intorstring)|An eviction is allowed if at most "maxUnavailable" pods selected by"selector" are unavailable after the eviction, i.e. even in absence ofthe evicted pod. For example, one can prevent all voluntary evictionsby specifying 0. This is a mutually exclusive setting with "minAvailable".|
|`minAvailable`|[`IntOrString`](#intorstring)|An eviction is allowed if at least "minAvailable" pods selected by"selector" will still be available after the eviction, i.e. even in theabsence of the evicted pod.  So for example you can prevent all voluntaryevictions by specifying "100%".|
|`selector`|[`LabelSelector`](#labelselector)|Label query over pods whose evictions are managed by the disruptionbudget.|

## PodSecurityContext

PodSecurityContext holds pod-level security attributes and common container settings.Some fields are also present in container.securityContext.  Field values ofcontainer.securityContext take precedence over field values of PodSecurityContext.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsGroup`|`int64`|1. The owning GID will be the FSGroup2. The setgid bit is set (new files created in the volume will be owned by FSGroup)3. The permission bits are OR'd with rw-rw----If unset, the Kubelet will not modify the ownership and permissions of any volume.|
|`runAsGroup`|`int64`|The GID to run the entrypoint of the container process.Uses runtime default if unset.May also be set in SecurityContext.  If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedencefor that container.|
|`runAsNonRoot`|`boolean`|Indicates that the container must run as a non-root user.If true, the Kubelet will validate the image at runtime to ensure that itdoes not run as UID 0 (root) and fail to start the container if it does.If unset or false, no such validation will be performed.May also be set in SecurityContext.  If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedence.|
|`runAsUser`|`int64`|The UID to run the entrypoint of the container process.Defaults to user specified in image metadata if unspecified.May also be set in SecurityContext.  If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedencefor that container.|
|`seLinuxOptions`|[`SELinuxOptions`](#selinuxoptions)|The SELinux context to be applied to all containers.If unspecified, the container runtime will allocate a random SELinux context for eachcontainer.  May also be set in SecurityContext.  If set inboth SecurityContext and PodSecurityContext, the value specified in SecurityContexttakes precedence for that container.|
|`supplementalGroups`|`Array< string >`|A list of groups applied to the first process run in each container, in additionto the container's primary GID.  If unspecified, no groups will be added toany container.|
|`sysctls`|`Array<`[`Sysctl`](#sysctl)`>`|Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupportedsysctls (by the container runtime) might fail to launch.|
|`windowsOptions`|[`WindowsSecurityContextOptions`](#windowssecuritycontextoptions)|The Windows specific settings applied to all containers.If unspecified, the options within a container's SecurityContext will be used.If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.|

## Toleration

The pod this Toleration is attached to tolerates any taint that matchesthe triple <key,value,effect> using the matching operator <operator>.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`effect`|`string`|Effect indicates the taint effect to match. Empty means match all taint effects.When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.|
|`key`|`string`|Key is the taint key that the toleration applies to. Empty means match all taint keys.If the key is empty, operator must be Exists; this combination means to match all values and all keys.|
|`operator`|`string`|Operator represents a key's relationship to the value.Valid operators are Exists and Equal. Defaults to Equal.Exists is equivalent to wildcard for value, so that a pod cantolerate all taints of a particular category.|
|`tolerationSeconds`|`int64`|TolerationSeconds represents the period of time the toleration (which must beof effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,it is not set, which means tolerate the taint forever (do not evict). Zero andnegative values will be treated as 0 (evict immediately) by the system.|
|`value`|`string`|Value is the taint value the toleration matches to.If the operator is Exists, the value should be empty, otherwise just a regular string.|

## PersistentVolumeClaim

PersistentVolumeClaim is a user's request for and claim to a persistent volume

<details>
<summary>Examples (click to open)</summary>
<br>

- [`testvolume.yaml`](../examples/testvolume.yaml)
</details>

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`metadata`|[`ObjectMeta`](#objectmeta)|Standard object's metadata.More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata|
|`spec`|[`PersistentVolumeClaimSpec`](#persistentvolumeclaimspec)|Spec defines the desired characteristics of a volume requested by a pod author.More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims|
|`status`|[`PersistentVolumeClaimStatus`](#persistentvolumeclaimstatus)|Status represents the current information/status of a persistent volume claim.Read-only.More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims|

## Volume

Volume represents a named volume in a pod that may be accessed by any container in the pod.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`init-container.yaml`](../examples/init-container.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Volume's name.Must be a DNS_LABEL and unique within the pod.More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`volumeSource`|[`VolumeSource`](#volumesource)|VolumeSource represents the location and type of the mounted volume.If not specified, the Volume is implied to be an EmptyDir.This implied behavior is deprecated and will be removed in a future version.|

## Time

Time is a wrapper around time.Time which supports correctmarshaling to YAML and JSON.  Wrappers are provided for manyof the factory methods that the time package offers.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nanos`|`int32`|Non-negative fractions of a second at nanosecond resolution. Negativesecond values with fractions must still have non-negative nanos valuesthat count forward in time. Must be from 0 to 999,999,999inclusive. This field may be limited in precision depending on context.|
|`seconds`|`int64`|Represents seconds of UTC time since Unix epoch1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to9999-12-31T23:59:59Z inclusive.|

## ObjectReference

ObjectReference contains enough information to let you inspect or modify the referred object.+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|API version of the referent.|
|`fieldPath`|`string`|If referring to a piece of an object instead of an entire object, this stringshould contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2].For example, if the object reference is to a container within a pod, this would take on a value like:"spec.containers{name}" (where "name" refers to the name of the container that triggeredthe event) or if no container name is specified "spec.containers[2]" (container withindex 2 in this pod). This syntax is chosen only to have some well-defined way ofreferencing a part of an object.TODO: this design is not final and this field is subject to change in the future.|
|`kind`|`string`|Kind of the referent.More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds|
|`name`|`string`|Name of the referent.More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names|
|`namespace`|`string`|Namespace of the referent.More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/|
|`resourceVersion`|`string`|Specific resourceVersion to which this reference is made, if any.More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency|
|`uid`|`string`|UID of the referent.More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids|

## Container

A single application container that you want to run within a pod.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](../examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`cron-workflow.yaml`](../examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`default-pdb-support.yaml`](../examples/default-pdb-support.yaml)

- [`dns-config.yaml`](../examples/dns-config.yaml)

- [`exit-handlers.yaml`](../examples/exit-handlers.yaml)

- [`forever.yaml`](../examples/forever.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](../examples/gc-ttl.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`hello-world.yaml`](../examples/hello-world.yaml)

- [`image-pull-secrets.yaml`](../examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`init-container.yaml`](../examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](../examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](../examples/retry-container.yaml)

- [`retry-on-error.yaml`](../examples/retry-on-error.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](../examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](../examples/sidecar.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)

- [`template-on-exit.yaml`](../examples/template-on-exit.yaml)

- [`timeouts-step.yaml`](../examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`args`|`Array< string >`|Arguments to the entrypoint.The docker image's CMD is used if this is not provided.Variable references $(VAR_NAME) are expanded using the container's environment. If a variablecannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntaxcan be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,regardless of whether the variable exists or not.Cannot be updated.More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell|
|`command`|`Array< string >`|Entrypoint array. Not executed within a shell.The docker image's ENTRYPOINT is used if this is not provided.Variable references $(VAR_NAME) are expanded using the container's environment. If a variablecannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntaxcan be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded,regardless of whether the variable exists or not.Cannot be updated.More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell|
|`env`|`Array<`[`EnvVar`](#envvar)`>`|List of environment variables to set in the container.Cannot be updated.|
|`envFrom`|`Array<`[`EnvFromSource`](#envfromsource)`>`|List of sources to populate environment variables in the container.The keys defined within a source must be a C_IDENTIFIER. All invalid keyswill be reported as an event when the container is starting. When a key exists in multiplesources, the value associated with the last source will take precedence.Values defined by an Env with a duplicate key will take precedence.Cannot be updated.|
|`image`|`string`|Docker image name.More info: https://kubernetes.io/docs/concepts/containers/imagesThis field is optional to allow higher level config management to default or overridecontainer images in workload controllers like Deployments and StatefulSets.|
|`imagePullPolicy`|`string`|Image pull policy.One of Always, Never, IfNotPresent.Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.Cannot be updated.More info: https://kubernetes.io/docs/concepts/containers/images#updating-images|
|`lifecycle`|[`Lifecycle`](#lifecycle)|Actions that the management system should take in response to container lifecycle events.Cannot be updated.|
|`livenessProbe`|[`Probe`](#probe)|Periodic probe of container liveness.Container will be restarted if the probe fails.Cannot be updated.More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`name`|`string`|Name of the container specified as a DNS_LABEL.Each container in a pod must have a unique name (DNS_LABEL).Cannot be updated.|
|`ports`|`Array<`[`ContainerPort`](#containerport)`>`|List of ports to expose from the container. Exposing a port here givesthe system additional information about the network connections acontainer uses, but is primarily informational. Not specifying a port hereDOES NOT prevent that port from being exposed. Any port which islistening on the default "0.0.0.0" address inside a container will beaccessible from the network.Cannot be updated.|
|`readinessProbe`|[`Probe`](#probe)|Periodic probe of container service readiness.Container will be removed from service endpoints if the probe fails.Cannot be updated.More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`resources`|[`ResourceRequirements`](#resourcerequirements)|Compute Resources required by this container.Cannot be updated.More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/|
|`securityContext`|[`SecurityContext`](#securitycontext)|Security options the pod should run with.More info: https://kubernetes.io/docs/concepts/policy/security-context/More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/|
|`startupProbe`|[`Probe`](#probe)|StartupProbe indicates that the Pod has successfully initialized.If specified, no other probes are executed until this completes successfully.If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,when it might take a long time to load data or warm a cache, than during steady-state operation.This cannot be updated.This is an alpha feature enabled by the StartupProbe feature flag.More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`stdin`|`boolean`|Whether this container should allocate a buffer for stdin in the container runtime. If thisis not set, reads from stdin in the container will always result in EOF.Default is false.|
|`stdinOnce`|`boolean`|Whether the container runtime should close the stdin channel after it has been opened bya single attach. When stdin is true the stdin stream will remain open across multiple attachsessions. If stdinOnce is set to true, stdin is opened on container start, is empty until thefirst client attaches to stdin, and then remains open and accepts data until the client disconnects,at which time stdin is closed and remains closed until the container is restarted. If thisflag is false, a container processes that reads from stdin will never receive an EOF.Default is false|
|`terminationMessagePath`|`string`|Optional: Path at which the file to which the container's termination messagewill be written is mounted into the container's filesystem.Message written is intended to be brief final status, such as an assertion failure message.Will be truncated by the node if greater than 4096 bytes. The total message length acrossall containers will be limited to 12kb.Defaults to /dev/termination-log.Cannot be updated.|
|`terminationMessagePolicy`|`string`|Indicate how the termination message should be populated. File will use the contents ofterminationMessagePath to populate the container status message on both success and failure.FallbackToLogsOnError will use the last chunk of container log output if the terminationmessage file is empty and the container exited with an error.The log output is limited to 2048 bytes or 80 lines, whichever is smaller.Defaults to File.Cannot be updated.|
|`tty`|`boolean`|Whether this container should allocate a TTY for itself, also requires 'stdin' to be true.Default is false.|
|`volumeDevices`|`Array<`[`VolumeDevice`](#volumedevice)`>`|volumeDevices is the list of block devices to be used by the container.This is a beta feature.|
|`volumeMounts`|`Array<`[`VolumeMount`](#volumemount)`>`|Pod volumes to mount into the container's filesystem.Cannot be updated.|
|`workingDir`|`string`|Container's working directory.If not specified, the container runtime's default will be used, whichmight be configured in the container image.Cannot be updated.|

## SecretKeySelector

SecretKeySelector selects a key of a Secret.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|The key of the secret to select from.  Must be a valid secret key.|
|`localObjectReference`|[`LocalObjectReference`](#localobjectreference)|The name of the secret in the pod's namespace to select from.|
|`optional`|`boolean`|Specify whether the Secret or its key must be defined|

## ConfigMapKeySelector

Selects a key from a ConfigMap.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|The key to select.|
|`localObjectReference`|[`LocalObjectReference`](#localobjectreference)|The ConfigMap to select from.|
|`optional`|`boolean`|Specify whether the ConfigMap or its key must be defined|

## ManagedFieldsEntry

ManagedFieldsEntry is a workflow-id, a FieldSet and the group version of the resourcethat the fieldset applies to.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|APIVersion defines the version of this resource that this field setapplies to. The format is "group/version" just like the top-levelAPIVersion field. It is necessary to track the version of a fieldset because it cannot be automatically converted.|
|`fieldsType`|`string`|FieldsType is the discriminator for the different fields format and version.There is currently only one possible value: "FieldsV1"|
|`fieldsV1`|[`FieldsV1`](#fieldsv1)|FieldsV1 holds the first JSON version format as described in the "FieldsV1" type.|
|`manager`|`string`|Manager is an identifier of the workflow managing these fields.|
|`operation`|`string`|Operation is the type of operation which lead to this ManagedFieldsEntry being created.The only valid values for this field are 'Apply' and 'Update'.|
|`time`|[`Time`](#time)|Time is timestamp of when these fields were set. It should always be empty if Operation is 'Apply'|

## OwnerReference

OwnerReference contains enough information to let you identify an owningobject. An owning object must be in the same namespace as the dependent, orbe cluster-scoped, so there is no namespace field.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`k8s-owner-reference.yaml`](../examples/k8s-owner-reference.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|API version of the referent.|
|`blockOwnerDeletion`|`boolean`|If true, AND if the owner has the "foregroundDeletion" finalizer, thenthe owner cannot be deleted from the key-value store until thisreference is removed.Defaults to false.To set this field, a user needs "delete" permission of the owner,otherwise 422 (Unprocessable Entity) will be returned.|
|`controller`|`boolean`|If true, this reference points to the managing controller.|
|`kind`|`string`|Kind of the referent.More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds|
|`name`|`string`|Name of the referent.More info: http://kubernetes.io/docs/user-guide/identifiers#names|
|`uid`|`string`|UID of the referent.More info: http://kubernetes.io/docs/user-guide/identifiers#uids|

## NodeAffinity

Node affinity is a group of node affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preferredDuringSchedulingIgnoredDuringExecution`|`Array<`[`PreferredSchedulingTerm`](#preferredschedulingterm)`>`|The scheduler will prefer to schedule pods to nodes that satisfythe affinity expressions specified by this field, but it may choosea node that violates one or more of the expressions. The node that ismost preferred is the one with the greatest sum of weights, i.e.for each node that meets all of the scheduling requirements (resourcerequest, requiredDuringScheduling affinity expressions, etc.),compute a sum by iterating through the elements of this field and adding"weight" to the sum if the node matches the corresponding matchExpressions; thenode(s) with the highest sum are the most preferred.|
|`requiredDuringSchedulingIgnoredDuringExecution`|[`NodeSelector`](#nodeselector)|If the affinity requirements specified by this field are not met atscheduling time, the pod will not be scheduled onto the node.If the affinity requirements specified by this field cease to be metat some point during pod execution (e.g. due to an update), the systemmay or may not try to eventually evict the pod from its node.|

## PodAffinity

Pod affinity is a group of inter pod affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preferredDuringSchedulingIgnoredDuringExecution`|`Array<`[`WeightedPodAffinityTerm`](#weightedpodaffinityterm)`>`|The scheduler will prefer to schedule pods to nodes that satisfythe affinity expressions specified by this field, but it may choosea node that violates one or more of the expressions. The node that ismost preferred is the one with the greatest sum of weights, i.e.for each node that meets all of the scheduling requirements (resourcerequest, requiredDuringScheduling affinity expressions, etc.),compute a sum by iterating through the elements of this field and adding"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; thenode(s) with the highest sum are the most preferred.|
|`requiredDuringSchedulingIgnoredDuringExecution`|`Array<`[`PodAffinityTerm`](#podaffinityterm)`>`|If the affinity requirements specified by this field are not met atscheduling time, the pod will not be scheduled onto the node.If the affinity requirements specified by this field cease to be metat some point during pod execution (e.g. due to a pod label update), thesystem may or may not try to eventually evict the pod from its node.When there are multiple elements, the lists of nodes corresponding to eachpodAffinityTerm are intersected, i.e. all terms must be satisfied.|

## PodAntiAffinity

Pod anti affinity is a group of inter pod anti affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preferredDuringSchedulingIgnoredDuringExecution`|`Array<`[`WeightedPodAffinityTerm`](#weightedpodaffinityterm)`>`|The scheduler will prefer to schedule pods to nodes that satisfythe anti-affinity expressions specified by this field, but it may choosea node that violates one or more of the expressions. The node that ismost preferred is the one with the greatest sum of weights, i.e.for each node that meets all of the scheduling requirements (resourcerequest, requiredDuringScheduling anti-affinity expressions, etc.),compute a sum by iterating through the elements of this field and adding"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; thenode(s) with the highest sum are the most preferred.|
|`requiredDuringSchedulingIgnoredDuringExecution`|`Array<`[`PodAffinityTerm`](#podaffinityterm)`>`|If the anti-affinity requirements specified by this field are not met atscheduling time, the pod will not be scheduled onto the node.If the anti-affinity requirements specified by this field cease to be metat some point during pod execution (e.g. due to a pod label update), thesystem may or may not try to eventually evict the pod from its node.When there are multiple elements, the lists of nodes corresponding to eachpodAffinityTerm are intersected, i.e. all terms must be satisfied.|

## PodDNSConfigOption

PodDNSConfigOption defines DNS resolver options of a pod.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`dns-config.yaml`](../examples/dns-config.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Required.|
|`value`|`string`||

## IntOrString



### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`intVal`|`int32`|_No description available_|
|`strVal`|`string`|_No description available_|
|`type`|`int64`|_No description available_|

## LabelSelector

A label selector is a label query over a set of resources. The result of matchLabels andmatchExpressions are ANDed. An empty label selector matches all objects. A nulllabel selector matches no objects.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`matchExpressions`|`Array<`[`LabelSelectorRequirement`](#labelselectorrequirement)`>`|matchExpressions is a list of label selector requirements. The requirements are ANDed.|
|`matchLabels`|`Map< string , string >`|matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabelsmap is equivalent to an element of matchExpressions, whose key field is "key", theoperator is "In", and the values array contains only "value". The requirements are ANDed.|

## SELinuxOptions

SELinuxOptions are the labels to be applied to the container

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`level`|`string`|Level is SELinux level label that applies to the container.|
|`role`|`string`|Role is a SELinux role label that applies to the container.|
|`type`|`string`|Type is a SELinux type label that applies to the container.|
|`user`|`string`|User is a SELinux user label that applies to the container.|

## Sysctl

Sysctl defines a kernel parameter to be set

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Name of a property to set|
|`value`|`string`|Value of a property to set|

## WindowsSecurityContextOptions

WindowsSecurityContextOptions contain Windows-specific options and credentials.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`gmsaCredentialSpec`|`string`|GMSACredentialSpec is where the GMSA admission webhook(https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of theGMSA credential spec named by the GMSACredentialSpecName field.This field is alpha-level and is only honored by servers that enable the WindowsGMSA feature flag.|
|`gmsaCredentialSpecName`|`string`|GMSACredentialSpecName is the name of the GMSA credential spec to use.This field is alpha-level and is only honored by servers that enable the WindowsGMSA feature flag.|
|`runAsUserName`|`string`|The UserName in Windows to run the entrypoint of the container process.Defaults to the user specified in image metadata if unspecified.May also be set in PodSecurityContext. If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedence.This field is alpha-level and it is only honored by servers that enable the WindowsRunAsUserName feature flag.|

## PersistentVolumeClaimSpec

PersistentVolumeClaimSpec describes the common attributes of storage devicesand allows a Source for provider-specific attributes

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`archive-location.yaml`](../examples/archive-location.yaml)

- [`arguments-artifacts.yaml`](../examples/arguments-artifacts.yaml)

- [`arguments-parameters.yaml`](../examples/arguments-parameters.yaml)

- [`artifact-disable-archive.yaml`](../examples/artifact-disable-archive.yaml)

- [`artifact-passing.yaml`](../examples/artifact-passing.yaml)

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`artifactory-artifact.yaml`](../examples/artifactory-artifact.yaml)

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`cluster-wftmpl-dag.yaml`](../examples/cluster-workflow-template/cluster-wftmpl-dag.yaml)

- [`clustertemplates.yaml`](../examples/cluster-workflow-template/clustertemplates.yaml)

- [`mixed-cluster-namespaced-wftmpl-steps.yaml`](../examples/cluster-workflow-template/mixed-cluster-namespaced-wftmpl-steps.yaml)

- [`coinflip-recursive.yaml`](../examples/coinflip-recursive.yaml)

- [`coinflip.yaml`](../examples/coinflip.yaml)

- [`conditionals.yaml`](../examples/conditionals.yaml)

- [`continue-on-fail.yaml`](../examples/continue-on-fail.yaml)

- [`cron-workflow.yaml`](../examples/cron-workflow.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-coinflip.yaml`](../examples/dag-coinflip.yaml)

- [`dag-continue-on-fail.yaml`](../examples/dag-continue-on-fail.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`dag-diamond-steps.yaml`](../examples/dag-diamond-steps.yaml)

- [`dag-diamond.yaml`](../examples/dag-diamond.yaml)

- [`dag-disable-failFast.yaml`](../examples/dag-disable-failFast.yaml)

- [`dag-multiroot.yaml`](../examples/dag-multiroot.yaml)

- [`dag-nested.yaml`](../examples/dag-nested.yaml)

- [`dag-targets.yaml`](../examples/dag-targets.yaml)

- [`default-pdb-support.yaml`](../examples/default-pdb-support.yaml)

- [`dns-config.yaml`](../examples/dns-config.yaml)

- [`exit-handlers.yaml`](../examples/exit-handlers.yaml)

- [`forever.yaml`](../examples/forever.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`gc-ttl.yaml`](../examples/gc-ttl.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`global-parameters.yaml`](../examples/global-parameters.yaml)

- [`hdfs-artifact.yaml`](../examples/hdfs-artifact.yaml)

- [`hello-world.yaml`](../examples/hello-world.yaml)

- [`image-pull-secrets.yaml`](../examples/image-pull-secrets.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`init-container.yaml`](../examples/init-container.yaml)

- [`input-artifact-gcs.yaml`](../examples/input-artifact-gcs.yaml)

- [`input-artifact-git.yaml`](../examples/input-artifact-git.yaml)

- [`input-artifact-http.yaml`](../examples/input-artifact-http.yaml)

- [`input-artifact-oss.yaml`](../examples/input-artifact-oss.yaml)

- [`input-artifact-raw.yaml`](../examples/input-artifact-raw.yaml)

- [`input-artifact-s3.yaml`](../examples/input-artifact-s3.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-owner-reference.yaml`](../examples/k8s-owner-reference.yaml)

- [`k8s-set-owner-reference.yaml`](../examples/k8s-set-owner-reference.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`loops-dag.yaml`](../examples/loops-dag.yaml)

- [`loops-maps.yaml`](../examples/loops-maps.yaml)

- [`loops-param-argument.yaml`](../examples/loops-param-argument.yaml)

- [`loops-param-result.yaml`](../examples/loops-param-result.yaml)

- [`loops-sequence.yaml`](../examples/loops-sequence.yaml)

- [`loops.yaml`](../examples/loops.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`node-selector.yaml`](../examples/node-selector.yaml)

- [`output-artifact-s3.yaml`](../examples/output-artifact-s3.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parallelism-limit.yaml`](../examples/parallelism-limit.yaml)

- [`parallelism-nested-dag.yaml`](../examples/parallelism-nested-dag.yaml)

- [`parallelism-nested-workflow.yaml`](../examples/parallelism-nested-workflow.yaml)

- [`parallelism-nested.yaml`](../examples/parallelism-nested.yaml)

- [`parallelism-template-limit.yaml`](../examples/parallelism-template-limit.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation-script.yaml`](../examples/parameter-aggregation-script.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-gc-strategy.yaml`](../examples/pod-gc-strategy.yaml)

- [`pod-metadata.yaml`](../examples/pod-metadata.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-patch.yaml`](../examples/pod-spec-patch.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`resubmit.yaml`](../examples/resubmit.yaml)

- [`retry-backoff.yaml`](../examples/retry-backoff.yaml)

- [`retry-container-to-completion.yaml`](../examples/retry-container-to-completion.yaml)

- [`retry-container.yaml`](../examples/retry-container.yaml)

- [`retry-on-error.yaml`](../examples/retry-on-error.yaml)

- [`retry-script.yaml`](../examples/retry-script.yaml)

- [`retry-with-steps.yaml`](../examples/retry-with-steps.yaml)

- [`scripts-bash.yaml`](../examples/scripts-bash.yaml)

- [`scripts-javascript.yaml`](../examples/scripts-javascript.yaml)

- [`scripts-python.yaml`](../examples/scripts-python.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)

- [`sidecar-nginx.yaml`](../examples/sidecar-nginx.yaml)

- [`sidecar.yaml`](../examples/sidecar.yaml)

- [`status-reference.yaml`](../examples/status-reference.yaml)

- [`steps.yaml`](../examples/steps.yaml)

- [`suspend-template.yaml`](../examples/suspend-template.yaml)

- [`template-on-exit.yaml`](../examples/template-on-exit.yaml)

- [`testvolume.yaml`](../examples/testvolume.yaml)

- [`timeouts-step.yaml`](../examples/timeouts-step.yaml)

- [`timeouts-workflow.yaml`](../examples/timeouts-workflow.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)

- [`dag.yaml`](../examples/workflow-template/dag.yaml)

- [`hello-world.yaml`](../examples/workflow-template/hello-world.yaml)

- [`retry-with-steps.yaml`](../examples/workflow-template/retry-with-steps.yaml)

- [`steps.yaml`](../examples/workflow-template/steps.yaml)

- [`templates.yaml`](../examples/workflow-template/templates.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessModes`|`Array< string >`|AccessModes contains the desired access modes the volume should have.More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1|
|`dataSource`|[`TypedLocalObjectReference`](#typedlocalobjectreference)|This field requires the VolumeSnapshotDataSource alpha feature gate to beenabled and currently VolumeSnapshot is the only supported data source.If the provisioner can support VolumeSnapshot data source, it will createa new volume and data will be restored to the volume at the same time.If the provisioner does not support VolumeSnapshot data source, volume willnot be created and the failure will be reported as an event.In the future, we plan to support more data source types and the behaviorof the provisioner may change.|
|`resources`|[`ResourceRequirements`](#resourcerequirements)|Resources represents the minimum resources the volume should have.More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources|
|`selector`|[`LabelSelector`](#labelselector)|A label query over volumes to consider for binding.|
|`storageClassName`|`string`|Name of the StorageClass required by the claim.More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1|
|`volumeMode`|`string`|volumeMode defines what type of volume is required by the claim.Value of Filesystem is implied when not included in claim spec.This is a beta feature.|
|`volumeName`|`string`|VolumeName is the binding reference to the PersistentVolume backing this claim.|

## PersistentVolumeClaimStatus

PersistentVolumeClaimStatus is the current status of a persistent volume claim.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessModes`|`Array< string >`|AccessModes contains the actual access modes the volume backing the PVC has.More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1|
|`capacity`|[`Quantity`](#quantity)|Represents the actual resources of the underlying volume.|
|`conditions`|`Array<`[`PersistentVolumeClaimCondition`](#persistentvolumeclaimcondition)`>`|Current Condition of persistent volume claim. If underlying persistent volume is beingresized then the Condition will be set to 'ResizeStarted'.|
|`phase`|`string`|Phase represents the current phase of PersistentVolumeClaim.|

## VolumeSource

Represents the source of a volume to mount.Only one of its members may be specified.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`awsElasticBlockStore`|[`AWSElasticBlockStoreVolumeSource`](#awselasticblockstorevolumesource)|AWSElasticBlockStore represents an AWS Disk resource that is attached to akubelet's host machine and then exposed to the pod.More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore|
|`azureDisk`|[`AzureDiskVolumeSource`](#azurediskvolumesource)|AzureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.|
|`azureFile`|[`AzureFileVolumeSource`](#azurefilevolumesource)|AzureFile represents an Azure File Service mount on the host and bind mount to the pod.|
|`cephfs`|[`CephFSVolumeSource`](#cephfsvolumesource)|CephFS represents a Ceph FS mount on the host that shares a pod's lifetime|
|`cinder`|[`CinderVolumeSource`](#cindervolumesource)|Cinder represents a cinder volume attached and mounted on kubelets host machine.More info: https://examples.k8s.io/mysql-cinder-pd/README.md|
|`configMap`|[`ConfigMapVolumeSource`](#configmapvolumesource)|ConfigMap represents a configMap that should populate this volume|
|`csi`|[`CSIVolumeSource`](#csivolumesource)|CSI (Container Storage Interface) represents storage that is handled by an external CSI driver (Alpha feature).|
|`downwardAPI`|[`DownwardAPIVolumeSource`](#downwardapivolumesource)|DownwardAPI represents downward API about the pod that should populate this volume|
|`emptyDir`|[`EmptyDirVolumeSource`](#emptydirvolumesource)|EmptyDir represents a temporary directory that shares a pod's lifetime.More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir|
|`fc`|[`FCVolumeSource`](#fcvolumesource)|FC represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.|
|`flexVolume`|[`FlexVolumeSource`](#flexvolumesource)|FlexVolume represents a generic volume resource that isprovisioned/attached using an exec based plugin.|
|`flocker`|[`FlockerVolumeSource`](#flockervolumesource)|Flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running|
|`gcePersistentDisk`|[`GCEPersistentDiskVolumeSource`](#gcepersistentdiskvolumesource)|GCEPersistentDisk represents a GCE Disk resource that is attached to akubelet's host machine and then exposed to the pod.More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk|
|~`gitRepo`~|~[`GitRepoVolumeSource`](#gitrepovolumesource)~|~GitRepo represents a git repository at a particular revision.~ DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount anEmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDirinto the Pod's container.|
|`glusterfs`|[`GlusterfsVolumeSource`](#glusterfsvolumesource)|Glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime.More info: https://examples.k8s.io/volumes/glusterfs/README.md|
|`hostPath`|[`HostPathVolumeSource`](#hostpathvolumesource)|HostPath represents a pre-existing file or directory on the hostmachine that is directly exposed to the container. This is generallyused for system agents or other privileged things that are allowedto see the host machine. Most containers will NOT need this.More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath---TODO(jonesdl) We need to restrict who can use host directory mounts and who can/can notmount host directories as read/write.|
|`iscsi`|[`ISCSIVolumeSource`](#iscsivolumesource)|ISCSI represents an ISCSI Disk resource that is attached to akubelet's host machine and then exposed to the pod.More info: https://examples.k8s.io/volumes/iscsi/README.md|
|`nfs`|[`NFSVolumeSource`](#nfsvolumesource)|NFS represents an NFS mount on the host that shares a pod's lifetimeMore info: https://kubernetes.io/docs/concepts/storage/volumes#nfs|
|`persistentVolumeClaim`|[`PersistentVolumeClaimVolumeSource`](#persistentvolumeclaimvolumesource)|PersistentVolumeClaimVolumeSource represents a reference to aPersistentVolumeClaim in the same namespace.More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims|
|`photonPersistentDisk`|[`PhotonPersistentDiskVolumeSource`](#photonpersistentdiskvolumesource)|PhotonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine|
|`portworxVolume`|[`PortworxVolumeSource`](#portworxvolumesource)|PortworxVolume represents a portworx volume attached and mounted on kubelets host machine|
|`projected`|[`ProjectedVolumeSource`](#projectedvolumesource)|Items for all in one resources secrets, configmaps, and downward API|
|`quobyte`|[`QuobyteVolumeSource`](#quobytevolumesource)|Quobyte represents a Quobyte mount on the host that shares a pod's lifetime|
|`rbd`|[`RBDVolumeSource`](#rbdvolumesource)|RBD represents a Rados Block Device mount on the host that shares a pod's lifetime.More info: https://examples.k8s.io/volumes/rbd/README.md|
|`scaleIO`|[`ScaleIOVolumeSource`](#scaleiovolumesource)|ScaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes.|
|`secret`|[`SecretVolumeSource`](#secretvolumesource)|Secret represents a secret that should populate this volume.More info: https://kubernetes.io/docs/concepts/storage/volumes#secret|
|`storageos`|[`StorageOSVolumeSource`](#storageosvolumesource)|StorageOS represents a StorageOS volume attached and mounted on Kubernetes nodes.|
|`vsphereVolume`|[`VsphereVirtualDiskVolumeSource`](#vspherevirtualdiskvolumesource)|VsphereVolume represents a vSphere volume attached and mounted on kubelets host machine|

## EnvVar

EnvVar represents an environment variable present in a Container.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`secrets.yaml`](../examples/secrets.yaml)

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|Name of the environment variable. Must be a C_IDENTIFIER.|
|`value`|`string`|Variable references $(VAR_NAME) are expandedusing the previous defined environment variables in the container andany service environment variables. If a variable cannot be resolved,the reference in the input string will be unchanged. The $(VAR_NAME)syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escapedreferences will never be expanded, regardless of whether the variableexists or not.Defaults to "".|
|`valueFrom`|[`EnvVarSource`](#envvarsource)|Source for the environment variable's value. Cannot be used if value is not empty.|

## EnvFromSource

EnvFromSource represents the source of a set of ConfigMaps

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMapRef`|[`ConfigMapEnvSource`](#configmapenvsource)|The ConfigMap to select from|
|`prefix`|`string`|An optional identifier to prepend to each key in the ConfigMap. Must be a C_IDENTIFIER.|
|`secretRef`|[`SecretEnvSource`](#secretenvsource)|The Secret to select from|

## Lifecycle

Lifecycle describes actions that the management system should take in response to container lifecycleevents. For the PostStart and PreStop lifecycle handlers, management of the container blocksuntil the action is complete, unless the container process fails, in which case the handler is aborted.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`postStart`|[`Handler`](#handler)|PostStart is called immediately after a container is created. If the handler fails,the container is terminated and restarted according to its restart policy.Other management of the container blocks until the hook completes.More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks|
|`preStop`|[`Handler`](#handler)|PreStop is called immediately before a container is terminated due to anAPI request or management event such as liveness/startup probe failure,preemption, resource contention, etc. The handler is not called if thecontainer crashes or exits. The reason for termination is passed to thehandler. The Pod's termination grace period countdown begins before thePreStop hooked is executed. Regardless of the outcome of the handler, thecontainer will eventually terminate within the Pod's termination graceperiod. Other management of the container blocks until the hook completesor until the termination grace period is reached.More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks|

## Probe

Probe describes a health check to be performed against a container to determine whether it isalive or ready to receive traffic.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`failureThreshold`|`int32`|Minimum consecutive failures for the probe to be considered failed after having succeeded.Defaults to 3. Minimum value is 1.|
|`handler`|[`Handler`](#handler)|The action taken to determine the health of a container|
|`initialDelaySeconds`|`int32`|Number of seconds after the container has started before liveness probes are initiated.More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|
|`periodSeconds`|`int32`|How often (in seconds) to perform the probe.Default to 10 seconds. Minimum value is 1.|
|`successThreshold`|`int32`|Minimum consecutive successes for the probe to be considered successful after having failed.Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.|
|`timeoutSeconds`|`int32`|Number of seconds after which the probe times out.Defaults to 1 second. Minimum value is 1.More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes|

## ContainerPort

ContainerPort represents a network port in a single container.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`containerPort`|`int32`|Number of port to expose on the pod's IP address.This must be a valid port number, 0 < x < 65536.|
|`hostIP`|`string`|What host IP to bind the external port to.|
|`hostPort`|`int32`|Number of port to expose on the host.If specified, this must be a valid port number, 0 < x < 65536.If HostNetwork is specified, this must match ContainerPort.Most containers do not need this.|
|`name`|`string`|If specified, this must be an IANA_SVC_NAME and unique within the pod. Eachnamed port in a pod must have a unique name. Name for the port that can bereferred to by services.|
|`protocol`|`string`|Protocol for port. Must be UDP, TCP, or SCTP.Defaults to "TCP".|

## ResourceRequirements

ResourceRequirements describes the compute resource requirements.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`dns-config.yaml`](../examples/dns-config.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)

- [`pod-spec-patch-wf-tmpl.yaml`](../examples/pod-spec-patch-wf-tmpl.yaml)

- [`pod-spec-yaml-patch.yaml`](../examples/pod-spec-yaml-patch.yaml)

- [`testvolume.yaml`](../examples/testvolume.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`limits`|[`Quantity`](#quantity)|Limits describes the maximum amount of compute resources allowed.More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/|
|`requests`|[`Quantity`](#quantity)|Requests describes the minimum amount of compute resources required.If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,otherwise to an implementation-defined value.More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/|

## SecurityContext

SecurityContext holds security configuration that will be applied to a container.Some fields are present in both SecurityContext and PodSecurityContext.  When bothare set, the values in SecurityContext take precedence.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`sidecar-dind.yaml`](../examples/sidecar-dind.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`allowPrivilegeEscalation`|`boolean`|AllowPrivilegeEscalation controls whether a process can gain moreprivileges than its parent process. This bool directly controls ifthe no_new_privs flag will be set on the container process.AllowPrivilegeEscalation is true always when the container is:1) run as Privileged2) has CAP_SYS_ADMIN|
|`capabilities`|[`Capabilities`](#capabilities)|The capabilities to add/drop when running containers.Defaults to the default set of capabilities granted by the container runtime.|
|`privileged`|`boolean`|Run container in privileged mode.Processes in privileged containers are essentially equivalent to root on the host.Defaults to false.|
|`procMount`|`string`|procMount denotes the type of proc mount to use for the containers.The default is DefaultProcMount which uses the container runtime defaults forreadonly paths and masked paths.This requires the ProcMountType feature flag to be enabled.|
|`readOnlyRootFilesystem`|`boolean`|Whether this container has a read-only root filesystem.Default is false.|
|`runAsGroup`|`int64`|The GID to run the entrypoint of the container process.Uses runtime default if unset.May also be set in PodSecurityContext.  If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedence.|
|`runAsNonRoot`|`boolean`|Indicates that the container must run as a non-root user.If true, the Kubelet will validate the image at runtime to ensure that itdoes not run as UID 0 (root) and fail to start the container if it does.If unset or false, no such validation will be performed.May also be set in PodSecurityContext.  If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedence.|
|`runAsUser`|`int64`|The UID to run the entrypoint of the container process.Defaults to user specified in image metadata if unspecified.May also be set in PodSecurityContext.  If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedence.|
|`seLinuxOptions`|[`SELinuxOptions`](#selinuxoptions)|The SELinux context to be applied to the container.If unspecified, the container runtime will allocate a random SELinux context for eachcontainer.  May also be set in PodSecurityContext.  If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedence.|
|`windowsOptions`|[`WindowsSecurityContextOptions`](#windowssecuritycontextoptions)|The Windows specific settings applied to all containers.If unspecified, the options from the PodSecurityContext will be used.If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.|

## VolumeDevice

volumeDevice describes a mapping of a raw block device within a container.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`devicePath`|`string`|devicePath is the path inside of the container that the device will be mapped to.|
|`name`|`string`|name must match the name of a persistentVolumeClaim in the pod|

## VolumeMount

VolumeMount describes a mounting of a Volume within a container.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`ci-output-artifact.yaml`](../examples/ci-output-artifact.yaml)

- [`ci.yaml`](../examples/ci.yaml)

- [`fun-with-gifs.yaml`](../examples/fun-with-gifs.yaml)

- [`init-container.yaml`](../examples/init-container.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)

- [`volumes-pvc.yaml`](../examples/volumes-pvc.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`mountPath`|`string`|Path within the container at which the volume should be mounted.  Mustnot contain ':'.|
|`mountPropagation`|`string`|mountPropagation determines how mounts are propagated from the hostto container and the other way around.When not set, MountPropagationNone is used.This field is beta in 1.10.|
|`name`|`string`|This must match the Name of a Volume.|
|`readOnly`|`boolean`|Mounted read-only if true, read-write otherwise (false or unspecified).Defaults to false.|
|`subPath`|`string`|Path within the volume from which the container's volume should be mounted.Defaults to "" (volume's root).|
|`subPathExpr`|`string`|Expanded path within the volume from which the container's volume should be mounted.Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment.Defaults to "" (volume's root).SubPathExpr and SubPath are mutually exclusive.This field is beta in 1.15.|

## FieldsV1

FieldsV1 stores a set of fields in a data structure like a Trie, in JSON format.Each key is either a '.' representing the field itself, and will always map to an empty set,or a string representing a sub-field or item. The string will follow one of these four formats:'f:<name>', where <name> is the name of a field in a struct, or key in a map'v:<value>', where <value> is the exact json formatted value of a list item'i:<index>', where <index> is position of a item in a list'k:<keys>', where <keys> is a map of  a list item's key fields to their unique valuesIf a key maps to an empty Fields value, the field that key represents is part of the set.The exact format is defined in sigs.k8s.io/structured-merge-diff

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`Raw`|`byte`|Raw is the underlying serialization of this object.|

## PreferredSchedulingTerm

An empty preferred scheduling term matches all objects with implicit weight 0(i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preference`|[`NodeSelectorTerm`](#nodeselectorterm)|A node selector term, associated with the corresponding weight.|
|`weight`|`int32`|Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.|

## NodeSelector

A node selector represents the union of the results of one or more label queriesover a set of nodes; that is, it represents the OR of the selectors representedby the node selector terms.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nodeSelectorTerms`|`Array<`[`NodeSelectorTerm`](#nodeselectorterm)`>`|Required. A list of node selector terms. The terms are ORed.|

## WeightedPodAffinityTerm

The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`podAffinityTerm`|[`PodAffinityTerm`](#podaffinityterm)|Required. A pod affinity term, associated with the corresponding weight.|
|`weight`|`int32`|weight associated with matching the corresponding podAffinityTerm,in the range 1-100.|

## PodAffinityTerm

Defines a set of pods (namely those matching the labelSelectorrelative to the given namespace(s)) that this pod should beco-located (affinity) or not co-located (anti-affinity) with,where co-located is defined as running on a node whose value ofthe label with key <topologyKey> matches that of any node on whicha pod of the set of pods is running

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`labelSelector`|[`LabelSelector`](#labelselector)|A label query over a set of resources, in this case pods.|
|`namespaces`|`Array< string >`|namespaces specifies which namespaces the labelSelector applies to (matches against);null or empty list means "this pod's namespace"|
|`topologyKey`|`string`|This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matchingthe labelSelector in the specified namespaces, where co-located is defined as running on a nodewhose value of the label with key topologyKey matches that of any node on which any of theselected pods is running.Empty topologyKey is not allowed.|

## LabelSelectorRequirement

A label selector requirement is a selector that contains values, a key, and an operator thatrelates the key and values.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|key is the label key that the selector applies to.|
|`operator`|`string`|operator represents a key's relationship to a set of values.Valid operators are In, NotIn, Exists and DoesNotExist.|
|`values`|`Array< string >`|values is an array of string values. If the operator is In or NotIn,the values array must be non-empty. If the operator is Exists or DoesNotExist,the values array must be empty. This array is replaced during a strategicmerge patch.|

## TypedLocalObjectReference

TypedLocalObjectReference contains enough information to let you locate thetyped referenced object inside the same namespace.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiGroup`|`string`|APIGroup is the group for the resource being referenced.If APIGroup is not specified, the specified Kind must be in the core API group.For any other third-party types, APIGroup is required.|
|`kind`|`string`|Kind is the type of resource being referenced|
|`name`|`string`|Name is the name of resource being referenced|

## Quantity

Quantity is a fixed-point representation of a number.It provides convenient marshaling/unmarshaling in JSON and YAML,in addition to String() and AsInt64() accessors.The serialization format is:<quantity>        ::= <signedNumber><suffix>  (Note that <suffix> may be empty, from the "" case in <decimalSI>.)<digit>           ::= 0 | 1 | ... | 9<digits>          ::= <digit> | <digit><digits><number>          ::= <digits> | <digits>.<digits> | <digits>. | .<digits><sign>            ::= "+" | "-"<signedNumber>    ::= <number> | <sign><number><suffix>          ::= <binarySI> | <decimalExponent> | <decimalSI><binarySI>        ::= Ki | Mi | Gi | Ti | Pi | Ei  (International System of units; See: http://physics.nist.gov/cuu/Units/binary.html)<decimalSI>       ::= m | "" | k | M | G | T | P | E  (Note that 1024 = 1Ki but 1000 = 1k; I didn't choose the capitalization.)<decimalExponent> ::= "e" <signedNumber> | "E" <signedNumber>No matter which of the three exponent forms is used, no quantity may representa number greater than 2^63-1 in magnitude, nor may it have more than 3 decimalplaces. Numbers larger or more precise will be capped or rounded up.(E.g.: 0.1m will rounded up to 1m.)This may be extended in the future if we require larger or smaller quantities.When a Quantity is parsed from a string, it will remember the type of suffixit had, and will use the same type again when it is serialized.Before serializing, Quantity will be put in "canonical form".This means that Exponent/suffix will be adjusted up or down (with acorresponding increase or decrease in Mantissa) such that:  a. No precision is lost  b. No fractional digits will be emitted  c. The exponent (or suffix) is as large as possible.The sign will be omitted unless the number is negative.Examples:  1.5 will be serialized as "1500m"  1.5Gi will be serialized as "1536Mi"Note that the quantity will NEVER be internally represented by afloating point number. That is the whole point of this exercise.Non-canonical values will still parse as long as they are well formed,but will be re-emitted in their canonical form. (So always use canonicalform, or don't diff.)This format is intended to make it difficult to use these numbers withoutwriting some sort of special handling code in the hopes that that willcause implementors to also use a fixed point implementation.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`string`|`string`|_No description available_|

## PersistentVolumeClaimCondition

PersistentVolumeClaimCondition contails details about state of pvc

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`lastProbeTime`|[`Time`](#time)|Last time we probed the condition.|
|`lastTransitionTime`|[`Time`](#time)|Last time the condition transitioned from one status to another.|
|`message`|`string`|Human-readable message indicating details about last transition.|
|`reason`|`string`|Unique, this should be a short, machine understandable string that gives the reasonfor condition's last transition. If it reports "ResizeStarted" that means the underlyingpersistent volume is being resized.|
|`status`|`string`|_No description available_|
|`type`|`string`|_No description available_|

## AWSElasticBlockStoreVolumeSource

Represents a Persistent Disk resource in AWS.An AWS EBS disk must exist before mounting to a container. The diskmust also be in the same AWS zone as the kubelet. An AWS EBS diskcan only be mounted as read/write once. AWS EBS volumes supportownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type of the volume that you want to mount.Tip: Ensure that the filesystem type is supported by the host operating system.Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstoreTODO: how do we prevent errors in the filesystem from compromising the machine|
|`partition`|`int32`|The partition in the volume that you want to mount.If omitted, the default is to mount by volume name.Examples: For volume /dev/sda1, you specify the partition as "1".Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty).|
|`readOnly`|`boolean`|Specify "true" to force and set the ReadOnly property in VolumeMounts to "true".If omitted, the default is "false".More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore|
|`volumeID`|`string`|Unique ID of the persistent disk resource in AWS (Amazon EBS volume).More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore|

## AzureDiskVolumeSource

AzureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`cachingMode`|`string`|Host Caching mode: None, Read Only, Read Write.|
|`diskName`|`string`|The Name of the data disk in the blob storage|
|`diskURI`|`string`|The URI the data disk in the blob storage|
|`fsType`|`string`|Filesystem type to mount.Must be a filesystem type supported by the host operating system.Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.|
|`kind`|`string`|Expected values Shared: multiple blob disks per storage account  Dedicated: single blob disk per storage account  Managed: azure managed data disk (only in managed availability set). defaults to shared|
|`readOnly`|`boolean`|Defaults to false (read/write). ReadOnly here will forcethe ReadOnly setting in VolumeMounts.|

## AzureFileVolumeSource

AzureFile represents an Azure File Service mount on the host and bind mount to the pod.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`readOnly`|`boolean`|Defaults to false (read/write). ReadOnly here will forcethe ReadOnly setting in VolumeMounts.|
|`secretName`|`string`|the name of secret that contains Azure Storage Account Name and Key|
|`shareName`|`string`|Share Name|

## CephFSVolumeSource

Represents a Ceph Filesystem mount that lasts the lifetime of a podCephfs volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`monitors`|`Array< string >`|Required: Monitors is a collection of Ceph monitorsMore info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it|
|`path`|`string`|Optional: Used as the mounted root, rather than the full Ceph tree, default is /|
|`readOnly`|`boolean`|Optional: Defaults to false (read/write). ReadOnly here will forcethe ReadOnly setting in VolumeMounts.More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it|
|`secretFile`|`string`|Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secretMore info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|Optional: SecretRef is reference to the authentication secret for User, default is empty.More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it|
|`user`|`string`|Optional: User is the rados user name, default is adminMore info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it|

## CinderVolumeSource

Represents a cinder volume resource in Openstack.A Cinder volume must exist before mounting to a container.The volume must also be in the same region as the kubelet.Cinder volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount.Must be a filesystem type supported by the host operating system.Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.More info: https://examples.k8s.io/mysql-cinder-pd/README.md|
|`readOnly`|`boolean`|Optional: Defaults to false (read/write). ReadOnly here will forcethe ReadOnly setting in VolumeMounts.More info: https://examples.k8s.io/mysql-cinder-pd/README.md|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|Optional: points to a secret object containing parameters used to connectto OpenStack.|
|`volumeID`|`string`|volume id used to identify the volume in cinder.More info: https://examples.k8s.io/mysql-cinder-pd/README.md|

## ConfigMapVolumeSource

Adapts a ConfigMap into a volume.The contents of the target ConfigMap's Data field will be presented in avolume as files using the keys in the Data field as the file names, unlessthe items element is populated with specific mappings of keys to paths.ConfigMap volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`defaultMode`|`int32`|Optional: mode bits to use on created files by default. Must be avalue between 0 and 0777. Defaults to 0644.Directories within the path are not affected by this setting.This might be in conflict with other options that affect the filemode, like fsGroup, and the result can be other mode bits set.|
|`items`|`Array<`[`KeyToPath`](#keytopath)`>`|If unspecified, each key-value pair in the Data field of the referencedConfigMap will be projected into the volume as a file whose name is thekey and content is the value. If specified, the listed keys will beprojected into the specified paths, and unlisted keys will not bepresent. If a key is specified which is not present in the ConfigMap,the volume setup will error unless it is marked optional. Paths must berelative and may not contain the '..' path or start with '..'.|
|`localObjectReference`|[`LocalObjectReference`](#localobjectreference)|_No description available_|
|`optional`|`boolean`|Specify whether the ConfigMap or its keys must be defined|

## CSIVolumeSource

Represents a source location of a volume to mount, managed by an external CSI driver

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`driver`|`string`|Driver is the name of the CSI driver that handles this volume.Consult with your admin for the correct name as registered in the cluster.|
|`fsType`|`string`|Filesystem type to mount. Ex. "ext4", "xfs", "ntfs".If not provided, the empty value is passed to the associated CSI driverwhich will determine the default filesystem to apply.|
|`nodePublishSecretRef`|[`LocalObjectReference`](#localobjectreference)|NodePublishSecretRef is a reference to the secret object containingsensitive information to pass to the CSI driver to complete the CSINodePublishVolume and NodeUnpublishVolume calls.This field is optional, and  may be empty if no secret is required. If thesecret object contains more than one secret, all secret references are passed.|
|`readOnly`|`boolean`|Specifies a read-only configuration for the volume.Defaults to false (read/write).|
|`volumeAttributes`|`Map< string , string >`|VolumeAttributes stores driver-specific properties that are passed to the CSIdriver. Consult your driver's documentation for supported values.|

## DownwardAPIVolumeSource

DownwardAPIVolumeSource represents a volume containing downward API io.argoproj.workflow.v1alpha1.Downward API volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`defaultMode`|`int32`|Optional: mode bits to use on created files by default. Must be avalue between 0 and 0777. Defaults to 0644.Directories within the path are not affected by this setting.This might be in conflict with other options that affect the filemode, like fsGroup, and the result can be other mode bits set.|
|`items`|`Array<`[`DownwardAPIVolumeFile`](#downwardapivolumefile)`>`|Items is a list of downward API volume file|

## EmptyDirVolumeSource

Represents an empty directory for a pod.Empty directory volumes support ownership management and SELinux relabeling.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`init-container.yaml`](../examples/init-container.yaml)

- [`volumes-emptydir.yaml`](../examples/volumes-emptydir.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`medium`|`string`|What type of storage medium should back this directory.The default is "" which means to use the node's default medium.Must be an empty string (default) or Memory.More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir|
|`sizeLimit`|[`Quantity`](#quantity)|Total amount of local storage required for this EmptyDir volume.The size limit is also applicable for memory medium.The maximum usage on memory medium EmptyDir would be the minimum value betweenthe SizeLimit specified here and the sum of memory limits of all containers in a pod.The default is nil which means that the limit is undefined.More info: http://kubernetes.io/docs/user-guide/volumes#emptydir|

## FCVolumeSource

Represents a Fibre Channel volume.Fibre Channel volumes can only be mounted as read/write once.Fibre Channel volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount.Must be a filesystem type supported by the host operating system.Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.TODO: how do we prevent errors in the filesystem from compromising the machine|
|`lun`|`int32`|Optional: FC target lun number|
|`readOnly`|`boolean`|Optional: Defaults to false (read/write). ReadOnly here will forcethe ReadOnly setting in VolumeMounts.|
|`targetWWNs`|`Array< string >`|Optional: FC target worldwide names (WWNs)|
|`wwids`|`Array< string >`|Optional: FC volume world wide identifiers (wwids)Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously.|

## FlexVolumeSource

FlexVolume represents a generic volume resource that isprovisioned/attached using an exec based plugin.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`driver`|`string`|Driver is the name of the driver to use for this volume.|
|`fsType`|`string`|Filesystem type to mount.Must be a filesystem type supported by the host operating system.Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script.|
|`options`|`Map< string , string >`|Optional: Extra command options if any.|
|`readOnly`|`boolean`|Optional: Defaults to false (read/write). ReadOnly here will forcethe ReadOnly setting in VolumeMounts.|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|Optional: SecretRef is reference to the secret object containingsensitive information to pass to the plugin scripts. This may beempty if no secret object is specified. If the secret objectcontains more than one secret, all secrets are passed to the pluginscripts.|

## FlockerVolumeSource

Represents a Flocker volume mounted by the Flocker agent.One and only one of datasetName and datasetUUID should be set.Flocker volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`datasetName`|`string`|Name of the dataset stored as metadata -> name on the dataset for Flockershould be considered as deprecated|
|`datasetUUID`|`string`|UUID of the dataset. This is unique identifier of a Flocker dataset|

## GCEPersistentDiskVolumeSource

Represents a Persistent Disk resource in Google Compute Engine.A GCE PD must exist before mounting to a container. The disk mustalso be in the same GCE project and zone as the kubelet. A GCE PDcan only be mounted as read/write once or read-only many times. GCEPDs support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type of the volume that you want to mount.Tip: Ensure that the filesystem type is supported by the host operating system.Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdiskTODO: how do we prevent errors in the filesystem from compromising the machine|
|`partition`|`int32`|The partition in the volume that you want to mount.If omitted, the default is to mount by volume name.Examples: For volume /dev/sda1, you specify the partition as "1".Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty).More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk|
|`pdName`|`string`|Unique name of the PD resource in GCE. Used to identify the disk in GCE.More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk|
|`readOnly`|`boolean`|ReadOnly here will force the ReadOnly setting in VolumeMounts.Defaults to false.More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk|

## GitRepoVolumeSource

Represents a volume that is populated with the contents of a git repository.Git repo volumes do not support ownership management.Git repo volumes support SELinux relabeling.DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount anEmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDirinto the Pod's container.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`directory`|`string`|Target directory name.Must not contain or start with '..'.  If '.' is supplied, the volume directory will be thegit repository.  Otherwise, if specified, the volume will contain the git repository inthe subdirectory with the given name.|
|`repository`|`string`|Repository URL|
|`revision`|`string`|Commit hash for the specified revision.|

## GlusterfsVolumeSource

Represents a Glusterfs mount that lasts the lifetime of a pod.Glusterfs volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`endpoints`|`string`|EndpointsName is the endpoint name that details Glusterfs topology.More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod|
|`path`|`string`|Path is the Glusterfs volume path.More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod|
|`readOnly`|`boolean`|ReadOnly here will force the Glusterfs volume to be mounted with read-only permissions.Defaults to false.More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod|

## HostPathVolumeSource

Represents a host path mapped into a pod.Host path volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`path`|`string`|Path of the directory on the host.If the path is a symlink, it will follow the link to the real path.More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath|
|`type`|`string`|Type for HostPath VolumeDefaults to ""More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath|

## ISCSIVolumeSource

Represents an ISCSI disk.ISCSI volumes can only be mounted as read/write once.ISCSI volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`chapAuthDiscovery`|`boolean`|whether support iSCSI Discovery CHAP authentication|
|`chapAuthSession`|`boolean`|whether support iSCSI Session CHAP authentication|
|`fsType`|`string`|Filesystem type of the volume that you want to mount.Tip: Ensure that the filesystem type is supported by the host operating system.Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsiTODO: how do we prevent errors in the filesystem from compromising the machine|
|`initiatorName`|`string`|Custom iSCSI Initiator Name.If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface<target portal>:<volume name> will be created for the connection.|
|`iqn`|`string`|Target iSCSI Qualified Name.|
|`iscsiInterface`|`string`|iSCSI Interface Name that uses an iSCSI transport.Defaults to 'default' (tcp).|
|`lun`|`int32`|iSCSI Target Lun number.|
|`portals`|`Array< string >`|iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the portis other than default (typically TCP ports 860 and 3260).|
|`readOnly`|`boolean`|ReadOnly here will force the ReadOnly setting in VolumeMounts.Defaults to false.|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|CHAP Secret for iSCSI target and initiator authentication|
|`targetPortal`|`string`|iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the portis other than default (typically TCP ports 860 and 3260).|

## NFSVolumeSource

Represents an NFS mount that lasts the lifetime of a pod.NFS volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`path`|`string`|Path that is exported by the NFS server.More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs|
|`readOnly`|`boolean`|ReadOnly here will forcethe NFS export to be mounted with read-only permissions.Defaults to false.More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs|
|`server`|`string`|Server is the hostname or IP address of the NFS server.More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs|

## PersistentVolumeClaimVolumeSource

PersistentVolumeClaimVolumeSource references the user's PVC in the same namespace.This volume finds the bound PV and mounts that volume for the pod. APersistentVolumeClaimVolumeSource is, essentially, a wrapper around anothertype of volume that is owned by someone else (the system).

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`volumes-existing.yaml`](../examples/volumes-existing.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`claimName`|`string`|ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume.More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims|
|`readOnly`|`boolean`|Will force the ReadOnly setting in VolumeMounts.Default false.|

## PhotonPersistentDiskVolumeSource

Represents a Photon Controller persistent disk resource.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount.Must be a filesystem type supported by the host operating system.Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.|
|`pdID`|`string`|ID that identifies Photon Controller persistent disk|

## PortworxVolumeSource

PortworxVolumeSource represents a Portworx volume resource.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|FSType represents the filesystem type to mountMust be a filesystem type supported by the host operating system.Ex. "ext4", "xfs". Implicitly inferred to be "ext4" if unspecified.|
|`readOnly`|`boolean`|Defaults to false (read/write). ReadOnly here will forcethe ReadOnly setting in VolumeMounts.|
|`volumeID`|`string`|VolumeID uniquely identifies a Portworx volume|

## ProjectedVolumeSource

Represents a projected volume source

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`defaultMode`|`int32`|Mode bits to use on created files by default. Must be a value between0 and 0777.Directories within the path are not affected by this setting.This might be in conflict with other options that affect the filemode, like fsGroup, and the result can be other mode bits set.|
|`sources`|`Array<`[`VolumeProjection`](#volumeprojection)`>`|list of volume projections|

## QuobyteVolumeSource

Represents a Quobyte mount that lasts the lifetime of a pod.Quobyte volumes do not support ownership management or SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`group`|`string`|Group to map volume access toDefault is no group|
|`readOnly`|`boolean`|ReadOnly here will force the Quobyte volume to be mounted with read-only permissions.Defaults to false.|
|`registry`|`string`|Registry represents a single or multiple Quobyte Registry servicesspecified as a string as host:port pair (multiple entries are separated with commas)which acts as the central registry for volumes|
|`tenant`|`string`|Tenant owning the given Quobyte volume in the BackendUsed with dynamically provisioned Quobyte volumes, value is set by the plugin|
|`user`|`string`|User to map volume access toDefaults to serivceaccount user|
|`volume`|`string`|Volume is a string that references an already created Quobyte volume by name.|

## RBDVolumeSource

Represents a Rados Block Device mount that lasts the lifetime of a pod.RBD volumes support ownership management and SELinux relabeling.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type of the volume that you want to mount.Tip: Ensure that the filesystem type is supported by the host operating system.Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.More info: https://kubernetes.io/docs/concepts/storage/volumes#rbdTODO: how do we prevent errors in the filesystem from compromising the machine|
|`image`|`string`|The rados image name.More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`keyring`|`string`|Keyring is the path to key ring for RBDUser.Default is /etc/ceph/keyring.More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`monitors`|`Array< string >`|A collection of Ceph monitors.More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`pool`|`string`|The rados pool name.Default is rbd.More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`readOnly`|`boolean`|ReadOnly here will force the ReadOnly setting in VolumeMounts.Defaults to false.More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|SecretRef is name of the authentication secret for RBDUser. If providedoverrides keyring.Default is nil.More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|
|`user`|`string`|The rados user name.Default is admin.More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it|

## ScaleIOVolumeSource

ScaleIOVolumeSource represents a persistent ScaleIO volume

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount.Must be a filesystem type supported by the host operating system.Ex. "ext4", "xfs", "ntfs".Default is "xfs".|
|`gateway`|`string`|The host address of the ScaleIO API Gateway.|
|`protectionDomain`|`string`|The name of the ScaleIO Protection Domain for the configured storage.|
|`readOnly`|`boolean`|Defaults to false (read/write). ReadOnly here will forcethe ReadOnly setting in VolumeMounts.|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|SecretRef references to the secret for ScaleIO user and othersensitive information. If this is not provided, Login operation will fail.|
|`sslEnabled`|`boolean`|Flag to enable/disable SSL communication with Gateway, default false|
|`storageMode`|`string`|Indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned.Default is ThinProvisioned.|
|`storagePool`|`string`|The ScaleIO Storage Pool associated with the protection domain.|
|`system`|`string`|The name of the storage system as configured in ScaleIO.|
|`volumeName`|`string`|The name of a volume already created in the ScaleIO systemthat is associated with this volume source.|

## SecretVolumeSource

Adapts a Secret into a volume.The contents of the target Secret's Data field will be presented in a volumeas files using the keys in the Data field as the file names.Secret volumes support ownership management and SELinux relabeling.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`secrets.yaml`](../examples/secrets.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`defaultMode`|`int32`|Optional: mode bits to use on created files by default. Must be avalue between 0 and 0777. Defaults to 0644.Directories within the path are not affected by this setting.This might be in conflict with other options that affect the filemode, like fsGroup, and the result can be other mode bits set.|
|`items`|`Array<`[`KeyToPath`](#keytopath)`>`|If unspecified, each key-value pair in the Data field of the referencedSecret will be projected into the volume as a file whose name is thekey and content is the value. If specified, the listed keys will beprojected into the specified paths, and unlisted keys will not bepresent. If a key is specified which is not present in the Secret,the volume setup will error unless it is marked optional. Paths must berelative and may not contain the '..' path or start with '..'.|
|`optional`|`boolean`|Specify whether the Secret or its keys must be defined|
|`secretName`|`string`|Name of the secret in the pod's namespace to use.More info: https://kubernetes.io/docs/concepts/storage/volumes#secret|

## StorageOSVolumeSource

Represents a StorageOS persistent volume resource.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount.Must be a filesystem type supported by the host operating system.Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.|
|`readOnly`|`boolean`|Defaults to false (read/write). ReadOnly here will forcethe ReadOnly setting in VolumeMounts.|
|`secretRef`|[`LocalObjectReference`](#localobjectreference)|SecretRef specifies the secret to use for obtaining the StorageOS APIcredentials.  If not specified, default values will be attempted.|
|`volumeName`|`string`|VolumeName is the human-readable name of the StorageOS volume.  Volumenames are only unique within a namespace.|
|`volumeNamespace`|`string`|VolumeNamespace specifies the scope of the volume within StorageOS.  If nonamespace is specified then the Pod's namespace will be used.  This allows theKubernetes name scoping to be mirrored within StorageOS for tighter integration.Set VolumeName to any name to override the default behaviour.Set to "default" if you are not using namespaces within StorageOS.Namespaces that do not pre-exist within StorageOS will be created.|

## VsphereVirtualDiskVolumeSource

Represents a vSphere volume resource.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsType`|`string`|Filesystem type to mount.Must be a filesystem type supported by the host operating system.Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.|
|`storagePolicyID`|`string`|Storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName.|
|`storagePolicyName`|`string`|Storage Policy Based Management (SPBM) profile name.|
|`volumePath`|`string`|Path that identifies vSphere volume vmdk|

## EnvVarSource

EnvVarSource represents a source for the value of an EnvVar.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`artifact-path-placeholders.yaml`](../examples/artifact-path-placeholders.yaml)

- [`custom-metrics.yaml`](../examples/custom-metrics.yaml)

- [`global-outputs.yaml`](../examples/global-outputs.yaml)

- [`k8s-jobs.yaml`](../examples/k8s-jobs.yaml)

- [`k8s-orchestration.yaml`](../examples/k8s-orchestration.yaml)

- [`k8s-wait-wf.yaml`](../examples/k8s-wait-wf.yaml)

- [`nested-workflow.yaml`](../examples/nested-workflow.yaml)

- [`output-parameter.yaml`](../examples/output-parameter.yaml)

- [`parameter-aggregation-dag.yaml`](../examples/parameter-aggregation-dag.yaml)

- [`parameter-aggregation.yaml`](../examples/parameter-aggregation.yaml)

- [`pod-spec-from-previous-step.yaml`](../examples/pod-spec-from-previous-step.yaml)

- [`secrets.yaml`](../examples/secrets.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMapKeyRef`|[`ConfigMapKeySelector`](#configmapkeyselector)|Selects a key of a ConfigMap.|
|`fieldRef`|[`ObjectFieldSelector`](#objectfieldselector)|Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations,spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP.|
|`resourceFieldRef`|[`ResourceFieldSelector`](#resourcefieldselector)|Selects a resource of the container: only resources limits and requests(limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.|
|`secretKeyRef`|[`SecretKeySelector`](#secretkeyselector)|Selects a key of a secret in the pod's namespace|

## ConfigMapEnvSource

ConfigMapEnvSource selects a ConfigMap to populate the environmentvariables with.The contents of the target ConfigMap's Data field will represent thekey-value pairs as environment variables.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`localObjectReference`|[`LocalObjectReference`](#localobjectreference)|The ConfigMap to select from.|
|`optional`|`boolean`|Specify whether the ConfigMap must be defined|

## SecretEnvSource

SecretEnvSource selects a Secret to populate the environmentvariables with.The contents of the target Secret's Data field will represent thekey-value pairs as environment variables.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`localObjectReference`|[`LocalObjectReference`](#localobjectreference)|The Secret to select from.|
|`optional`|`boolean`|Specify whether the Secret must be defined|

## Handler

Handler defines a specific action that should be takenTODO: pass structured data to these actions, and document that data here.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`exec`|[`ExecAction`](#execaction)|One and only one of the following should be specified.Exec specifies the action to take.|
|`httpGet`|[`HTTPGetAction`](#httpgetaction)|HTTPGet specifies the http request to perform.|
|`tcpSocket`|[`TCPSocketAction`](#tcpsocketaction)|TCPSocket specifies an action involving a TCP port.TCP hooks not yet supportedTODO: implement a realistic TCP lifecycle hook|

## Capabilities

Adds and removes POSIX capabilities from running containers.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`add`|`Array< string >`|Added capabilities|
|`drop`|`Array< string >`|Removed capabilities|

## NodeSelectorTerm

A null or empty node selector term matches no objects. The requirements ofthem are ANDed.The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`matchExpressions`|`Array<`[`NodeSelectorRequirement`](#nodeselectorrequirement)`>`|A list of node selector requirements by node's labels.|
|`matchFields`|`Array<`[`NodeSelectorRequirement`](#nodeselectorrequirement)`>`|A list of node selector requirements by node's fields.|

## KeyToPath

Maps a string key to a path within a volume.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|The key to project.|
|`mode`|`int32`|Optional: mode bits to use on this file, must be a value between 0and 0777. If not specified, the volume defaultMode will be used.This might be in conflict with other options that affect the filemode, like fsGroup, and the result can be other mode bits set.|
|`path`|`string`|The relative path of the file to map the key to.May not be an absolute path.May not contain the path element '..'.May not start with the string '..'.|

## DownwardAPIVolumeFile

DownwardAPIVolumeFile represents information to create the file containing the pod field

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fieldRef`|[`ObjectFieldSelector`](#objectfieldselector)|Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.|
|`mode`|`int32`|Optional: mode bits to use on this file, must be a value between 0and 0777. If not specified, the volume defaultMode will be used.This might be in conflict with other options that affect the filemode, like fsGroup, and the result can be other mode bits set.|
|`path`|`string`|Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'|
|`resourceFieldRef`|[`ResourceFieldSelector`](#resourcefieldselector)|Selects a resource of the container: only resources limits and requests(limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.|

## VolumeProjection

Projection that may be projected along with other supported volume types

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMap`|[`ConfigMapProjection`](#configmapprojection)|information about the configMap data to project|
|`downwardAPI`|[`DownwardAPIProjection`](#downwardapiprojection)|information about the downwardAPI data to project|
|`secret`|[`SecretProjection`](#secretprojection)|information about the secret data to project|
|`serviceAccountToken`|[`ServiceAccountTokenProjection`](#serviceaccounttokenprojection)|information about the serviceAccountToken data to project|

## ObjectFieldSelector

ObjectFieldSelector selects an APIVersioned field of an object.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`apiVersion`|`string`|Version of the schema the FieldPath is written in terms of, defaults to "v1".|
|`fieldPath`|`string`|Path of the field to select in the specified API version.|

## ResourceFieldSelector

ResourceFieldSelector represents container resources (cpu, memory) and their output format

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`containerName`|`string`|Container name: required for volumes, optional for env vars|
|`divisor`|[`Quantity`](#quantity)|Specifies the output format of the exposed resources, defaults to "1"|
|`resource`|`string`|Required: resource to select|

## ExecAction

ExecAction describes a "run in container" action.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`command`|`Array< string >`|Command is the command line to execute inside the container, the working directory for thecommand  is root ('/') in the container's filesystem. The command is simply exec'd, it isnot run inside a shell, so traditional shell instructions ('|', etc) won't work. To usea shell, you need to explicitly call out to that shell.Exit status of 0 is treated as live/healthy and non-zero is unhealthy.|

## HTTPGetAction

HTTPGetAction describes an action based on HTTP Get requests.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`daemon-nginx.yaml`](../examples/daemon-nginx.yaml)

- [`daemon-step.yaml`](../examples/daemon-step.yaml)

- [`dag-daemon-task.yaml`](../examples/dag-daemon-task.yaml)

- [`influxdb-ci.yaml`](../examples/influxdb-ci.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`host`|`string`|Host name to connect to, defaults to the pod IP. You probably want to set"Host" in httpHeaders instead.|
|`httpHeaders`|`Array<`[`HTTPHeader`](#httpheader)`>`|Custom headers to set in the request. HTTP allows repeated headers.|
|`path`|`string`|Path to access on the HTTP server.|
|`port`|[`IntOrString`](#intorstring)|Name or number of the port to access on the container.Number must be in the range 1 to 65535.Name must be an IANA_SVC_NAME.|
|`scheme`|`string`|Scheme to use for connecting to the host.Defaults to HTTP.|

## TCPSocketAction

TCPSocketAction describes an action based on opening a socket

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`host`|`string`|Optional: Host name to connect to, defaults to the pod IP.|
|`port`|[`IntOrString`](#intorstring)|Number or name of the port to access on the container.Number must be in the range 1 to 65535.Name must be an IANA_SVC_NAME.|

## NodeSelectorRequirement

A node selector requirement is a selector that contains values, a key, and an operatorthat relates the key and values.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|The label key that the selector applies to.|
|`operator`|`string`|Represents a key's relationship to a set of values.Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.|
|`values`|`Array< string >`|An array of string values. If the operator is In or NotIn,the values array must be non-empty. If the operator is Exists or DoesNotExist,the values array must be empty. If the operator is Gt or Lt, the valuesarray must have a single element, which will be interpreted as an integer.This array is replaced during a strategic merge patch.|

## ConfigMapProjection

Adapts a ConfigMap into a projected volume.The contents of the target ConfigMap's Data field will be presented in aprojected volume as files using the keys in the Data field as the file names,unless the items element is populated with specific mappings of keys to paths.Note that this is identical to a configmap volume source without the defaultmode.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`items`|`Array<`[`KeyToPath`](#keytopath)`>`|If unspecified, each key-value pair in the Data field of the referencedConfigMap will be projected into the volume as a file whose name is thekey and content is the value. If specified, the listed keys will beprojected into the specified paths, and unlisted keys will not bepresent. If a key is specified which is not present in the ConfigMap,the volume setup will error unless it is marked optional. Paths must berelative and may not contain the '..' path or start with '..'.|
|`localObjectReference`|[`LocalObjectReference`](#localobjectreference)|_No description available_|
|`optional`|`boolean`|Specify whether the ConfigMap or its keys must be defined|

## DownwardAPIProjection

Represents downward API info for projecting into a projected volume.Note that this is identical to a downwardAPI volume source without the defaultmode.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`items`|`Array<`[`DownwardAPIVolumeFile`](#downwardapivolumefile)`>`|Items is a list of DownwardAPIVolume file|

## SecretProjection

Adapts a secret into a projected volume.The contents of the target Secret's Data field will be presented in aprojected volume as files using the keys in the Data field as the file names.Note that this is identical to a secret volume source without the defaultmode.

<details>
<summary>Examples with this field (click to open)</summary>
<br>

- [`secrets.yaml`](../examples/secrets.yaml)
</details>

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`items`|`Array<`[`KeyToPath`](#keytopath)`>`|If unspecified, each key-value pair in the Data field of the referencedSecret will be projected into the volume as a file whose name is thekey and content is the value. If specified, the listed keys will beprojected into the specified paths, and unlisted keys will not bepresent. If a key is specified which is not present in the Secret,the volume setup will error unless it is marked optional. Paths must berelative and may not contain the '..' path or start with '..'.|
|`localObjectReference`|[`LocalObjectReference`](#localobjectreference)|_No description available_|
|`optional`|`boolean`|Specify whether the Secret or its key must be defined|

## ServiceAccountTokenProjection

ServiceAccountTokenProjection represents a projected service account tokenvolume. This projection can be used to insert a service account token intothe pods runtime filesystem for use against APIs (Kubernetes API Server orotherwise).

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`audience`|`string`|Audience is the intended audience of the token. A recipient of a tokenmust identify itself with an identifier specified in the audience of thetoken, and otherwise should reject the token. The audience defaults to theidentifier of the apiserver.|
|`expirationSeconds`|`int64`|ExpirationSeconds is the requested duration of validity of the serviceaccount token. As the token approaches expiration, the kubelet volumeplugin will proactively rotate the service account token. The kubelet willstart trying to rotate the token if the token is older than 80 percent ofits time to live or if the token is older than 24 hours.Defaults to 1 hourand must be at least 10 minutes.|
|`path`|`string`|Path is the path relative to the mount point of the file to project thetoken into.|

## HTTPHeader

HTTPHeader describes a custom header to be used in HTTP probes

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`name`|`string`|The header field name|
|`value`|`string`|The header field value|
