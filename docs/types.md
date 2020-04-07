
# Argo Types


## Workflow

Workflow is the definition of a workflow resource
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`metadata`|[`ObjectMeta`](#objectmeta)|ObjectMeta is metadata that all persisted resources must have, which includes all objectsusers must create.|
|`spec`|[`WorkflowSpec`](#workflowspec)|WorkflowSpec is the specification of a Workflow.|
|`status`|[`WorkflowStatus`](#workflowstatus)|WorkflowStatus contains overall status information about a workflow|

## WorkflowSpec

WorkflowSpec is the specification of a Workflow.
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`activeDeadlineSeconds`|`int64`|Optional duration in seconds relative to the workflow start time which the workflow isallowed to run before the controller terminates the io.argoproj.workflow.v1alpha1. A value of zero is used toterminate a Running workflow|
|`affinity`|[`Affinity`](#affinity)|Affinity is a group of affinity scheduling rules.|
|`arguments`|[`Arguments`](#arguments)|Arguments to a template|
|`artifactRepositoryRef`|[`ArtifactRepositoryRef`](#artifactrepositoryref)||
|`automountServiceAccountToken`|`boolean`|AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods.ServiceAccountName of ExecutorConfig must be specified if this value is false.|
|`dnsConfig`|[`PodDNSConfig`](#poddnsconfig)|PodDNSConfig defines the DNS parameters of a pod in addition tothose generated from DNSPolicy.|
|`dnsPolicy`|`string`|Set DNS policy for the pod.Defaults to "ClusterFirst".Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'.DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy.To have DNS options set along with hostNetwork, you have to specify DNS policyexplicitly to 'ClusterFirstWithHostNet'.|
|`entrypoint`|`string`|Entrypoint is a template reference to the starting point of the io.argoproj.workflow.v1alpha1.|
|`executor`|[`ExecutorConfig`](#executorconfig)|ExecutorConfig holds configurations of an executor container.|
|`hostAliases`|`Array<`[`HostAlias`](#hostalias)`>`||
|`hostNetwork`|`boolean`|Host networking requested for this workflow pod. Default to false.|
|`imagePullSecrets`|`Array<`[`LocalObjectReference`](#localobjectreference)`>`|ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any imagesin pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secretscan be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet.More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod|
|`metrics`|[`Metrics`](#metrics)|Metrics are a list of metrics emitted from a Workflow/Template|
|`nodeSelector`|`Map< string , string >`|NodeSelector is a selector which will result in all pods of the workflowto be scheduled on the selected node(s). This is able to be overridden bya nodeSelector specified in the template.|
|`onExit`|`string`|OnExit is a template reference which is invoked at the end of theworkflow, irrespective of the success, failure, or error of theprimary io.argoproj.workflow.v1alpha1.|
|`parallelism`|`int64`|Parallelism limits the max total parallel pods that can execute at the same time in a workflow|
|`podDisruptionBudget`|[`PodDisruptionBudgetSpec`](#poddisruptionbudgetspec)|PodDisruptionBudgetSpec is a description of a PodDisruptionBudget.|
|`podGC`|[`PodGC`](#podgc)|PodGC describes how to delete completed pods as they complete|
|`podPriority`|`int32`|Priority to apply to workflow pods.|
|`podPriorityClassName`|`string`|PriorityClassName to apply to workflow pods.|
|`podSpecPatch`|`string`|PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization ofcontainer fields which are not strings (e.g. resource limits).|
|`priority`|`int32`|Priority is used if controller is configured to process limited number of workflows in parallel. Workflows with higher priority are processed first.|
|`schedulerName`|`string`|Set scheduler name for all pods.Will be overridden if container/script template's scheduler name is set.Default scheduler will be used if neither specified.+optional|
|`securityContext`|[`PodSecurityContext`](#podsecuritycontext)|PodSecurityContext holds pod-level security attributes and common container settings.Some fields are also present in container.securityContext.  Field values ofcontainer.securityContext take precedence over field values of PodSecurityContext.|
|`serviceAccountName`|`string`|ServiceAccountName is the name of the ServiceAccount to run all pods of the workflow as.|
|`shutdown`|`string`|Shutdown will shutdown the workflow according to its ShutdownStrategy|
|`suspend`|`boolean`|Suspend will suspend the workflow and prevent execution of any future steps in the workflow|
|`templates`|`Array<`[`Template`](#template)`>`|Templates is a list of workflow templates used in a workflow|
|`tolerations`|`Array<`[`Toleration`](#toleration)`>`|Tolerations to apply to workflow pods.|
|`ttlSecondsAfterFinished`|`int32`|TTLSecondsAfterFinished limits the lifetime of a Workflow that has finished execution(Succeeded, Failed, Error). If this field is set, once the Workflow finishes, it will bedeleted after ttlSecondsAfterFinished expires. If this field is unset,ttlSecondsAfterFinished will not expire. If this field is set to zero,ttlSecondsAfterFinished expires immediately after the Workflow finishes.DEPRECATED: Use TTLStrategy.SecondsAfterCompletion instead.|
|`ttlStrategy`|[`TTLStrategy`](#ttlstrategy)|TTLStrategy is the strategy for the time to live depending on if the workflow succeded or failed|
|`volumeClaimTemplates`|`Array<`[`PersistentVolumeClaim`](#persistentvolumeclaim)`>`|VolumeClaimTemplates is a list of claims that containers are allowed to reference.The Workflow controller will create the claims at the beginning of the workflowand delete the claims upon completion of the workflow|
|`volumes`|`Array<`[`Volume`](#volume)`>`|Volumes is a list of volumes that can be mounted by containers in a io.argoproj.workflow.v1alpha1.|

## WorkflowStatus

WorkflowStatus contains overall status information about a workflow
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`compressedNodes`|`string`|Compressed and base64 decoded Nodes map|
|`conditions`|`Array<`[`WorkflowCondition`](#workflowcondition)`>`|Conditions is a list of conditions the Workflow may have|
|`finishedAt`|[`Time`](#time)|Time is a wrapper around time.Time which supports correctmarshaling to YAML and JSON.  Wrappers are provided for manyof the factory methods that the time package offers.|
|`message`|`string`|A human readable message indicating details about why the workflow is in this condition.|
|`nodes`|[`NodeStatus`](#nodestatus)|Nodes is a mapping between a node ID and the node's status.|
|`offloadNodeStatusVersion`|`string`|Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty.This will actually be populated with a hash of the offloaded data.|
|`outputs`|[`Outputs`](#outputs)|Outputs hold parameters, artifacts, and results from a step|
|`persistentVolumeClaims`|`Array<`[`Volume`](#volume)`>`|PersistentVolumeClaims tracks all PVCs that were created as part of the io.argoproj.workflow.v1alpha1.The contents of this list are drained at the end of the workflow.|
|`phase`|`string`|Phase a simple, high-level summary of where the workflow is in its lifecycle.|
|`resourcesDuration`|`Map< string , int64 >`|ResourcesDuration is the total for the workflow|
|`startedAt`|[`Time`](#time)|Time is a wrapper around time.Time which supports correctmarshaling to YAML and JSON.  Wrappers are provided for manyof the factory methods that the time package offers.|
|`storedTemplates`|[`Template`](#template)|StoredTemplates is a mapping between a template ref and the node's status.|

## Arguments

Arguments to a template
    
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
|`configMap`|`string`||
|`key`|`string`||

## ExecutorConfig

ExecutorConfig holds configurations of an executor container.
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`serviceAccountName`|`string`|ServiceAccountName specifies the service account name of the executor container.|

## Metrics

Metrics are a list of metrics emitted from a Workflow/Template
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`prometheus`|`Array<`[`Prometheus`](#prometheus)`>`|Prometheus is a list of prometheus metrics to be emitted|

## PodGC

PodGC describes how to delete completed pods as they complete
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`strategy`|`string`||

## Template

Template is a reusable and composable unit of execution in a workflow
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`activeDeadlineSeconds`|`int64`|Optional duration in seconds relative to the StartTime that the pod may be active on a nodebefore the system actively tries to terminate the pod; value must be positive integerThis field is only applicable to container and script templates.|
|`affinity`|[`Affinity`](#affinity)|Affinity is a group of affinity scheduling rules.|
|`archiveLocation`|[`ArtifactLocation`](#artifactlocation)|ArtifactLocation describes a location for a single or multiple artifacts.It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname).It is also used to describe the location of multiple artifacts such as the archive locationof a single workflow step, which the executor will use as a default location to store its files.|
|`arguments`|[`Arguments`](#arguments)|Arguments to a template|
|`automountServiceAccountToken`|`boolean`|AutomountServiceAccountToken indicates whether a service account token should be automatically mounted in pods.ServiceAccountName of ExecutorConfig must be specified if this value is false.|
|`container`|[`Container`](#container)|A single application container that you want to run within a pod.|
|`daemon`|`boolean`|Deamon will allow a workflow to proceed to the next step so long as the container reaches readiness|
|`dag`|[`DAGTemplate`](#dagtemplate)|DAGTemplate is a template subtype for directed acyclic graph templates|
|`executor`|[`ExecutorConfig`](#executorconfig)|ExecutorConfig holds configurations of an executor container.|
|`hostAliases`|`Array<`[`HostAlias`](#hostalias)`>`|HostAliases is an optional list of hosts and IPs that will be injected into the pod spec|
|`initContainers`|`Array<`[`UserContainer`](#usercontainer)`>`|InitContainers is a list of containers which run before the main container.|
|`inputs`|[`Inputs`](#inputs)|Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another|
|`metadata`|[`Metadata`](#metadata)|Pod metdata|
|`metrics`|[`Metrics`](#metrics)|Metrics are a list of metrics emitted from a Workflow/Template|
|`name`|`string`|Name is the name of the template|
|`nodeSelector`|`Map< string , string >`|NodeSelector is a selector to schedule this step of the workflow to berun on the selected node(s). Overrides the selector set at the workflow level.|
|`outputs`|[`Outputs`](#outputs)|Outputs hold parameters, artifacts, and results from a step|
|`parallelism`|`int64`|Parallelism limits the max total parallel pods that can execute at the same time within theboundaries of this template invocation. If additional steps/dag templates are invoked, thepods created by those templates will not be counted towards this total.|
|`podSpecPatch`|`string`|PodSpecPatch holds strategic merge patch to apply against the pod spec. Allows parameterization ofcontainer fields which are not strings (e.g. resource limits).|
|`priority`|`int32`|Priority to apply to workflow pods.|
|`priorityClassName`|`string`|PriorityClassName to apply to workflow pods.|
|`resource`|[`ResourceTemplate`](#resourcetemplate)|ResourceTemplate is a template subtype to manipulate kubernetes resources|
|`resubmitPendingPods`|`boolean`|ResubmitPendingPods is a flag to enable resubmitting pods that remain Pending after initial submission|
|`retryStrategy`|[`RetryStrategy`](#retrystrategy)|RetryStrategy provides controls on how to retry a workflow step|
|`schedulerName`|`string`|If specified, the pod will be dispatched by specified scheduler.Or it will be dispatched by workflow scope scheduler if specified.If neither specified, the pod will be dispatched by default scheduler.+optional|
|`script`|[`ScriptTemplate`](#scripttemplate)|ScriptTemplate is a template subtype to enable scripting through code steps|
|`securityContext`|[`PodSecurityContext`](#podsecuritycontext)|PodSecurityContext holds pod-level security attributes and common container settings.Some fields are also present in container.securityContext.  Field values ofcontainer.securityContext take precedence over field values of PodSecurityContext.|
|`serviceAccountName`|`string`|ServiceAccountName to apply to workflow pods|
|`sidecars`|`Array<`[`UserContainer`](#usercontainer)`>`|Sidecars is a list of containers which run alongside the main containerSidecars are automatically killed when the main container completes|
|`steps`|`Array<`[`ParallelSteps`](#parallelsteps)`>`|Steps define a series of sequential/parallel workflow steps|
|`suspend`|[`SuspendTemplate`](#suspendtemplate)|SuspendTemplate is a template subtype to suspend a workflow at a predetermined point in time|
|`template`|`string`|Template is the name of the template which is used as the base of this template.DEPRECATED: This field is not used.|
|`templateRef`|[`TemplateRef`](#templateref)|TemplateRef is a reference of template resource.|
|`tolerations`|`Array<`[`Toleration`](#toleration)`>`|Tolerations to apply to workflow pods.|
|`volumes`|`Array<`[`Volume`](#volume)`>`|Volumes is a list of volumes that can be mounted by containers in a template.|

## TTLStrategy

TTLStrategy is the strategy for the time to live depending on if the workflow succeded or failed
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`secondsAfterCompletion`|`int32`||
|`secondsAfterFailure`|`int32`||
|`secondsAfterSuccess`|`int32`||

## WorkflowCondition

_No description available_
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`message`|`string`||
|`status`|`string`||
|`type`|`string`||

## NodeStatus

NodeStatus contains status information about an individual node in the workflow
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`boundaryID`|`string`|BoundaryID indicates the node ID of the associated template root node in which this node belongs to|
|`children`|`Array< string >`|Children is a list of child node IDs|
|`daemoned`|`boolean`|Daemoned tracks whether or not this node was daemoned and need to be terminated|
|`displayName`|`string`|DisplayName is a human readable representation of the node. Unique within a template boundary|
|`finishedAt`|[`Time`](#time)|Time is a wrapper around time.Time which supports correctmarshaling to YAML and JSON.  Wrappers are provided for manyof the factory methods that the time package offers.|
|`id`|`string`|ID is a unique identifier of a node within the worklowIt is implemented as a hash of the node name, which makes the ID deterministic|
|`inputs`|[`Inputs`](#inputs)|Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another|
|`message`|`string`|A human readable message indicating details about why the node is in this condition.|
|`name`|`string`|Name is unique name in the node tree used to generate the node ID|
|`outboundNodes`|`Array< string >`|OutboundNodes tracks the node IDs which are considered "outbound" nodes to a template invocation.For every invocation of a template, there are nodes which we considered as "outbound". Essentially,these are last nodes in the execution sequence to run, before the template is considered completed.These nodes are then connected as parents to a following step.In the case of single pod steps (i.e. container, script, resource templates), this list will be nilsince the pod itself is already considered the "outbound" node.In the case of DAGs, outbound nodes are the "target" tasks (tasks with no children).In the case of steps, outbound nodes are all the containers involved in the last step group.NOTE: since templates are composable, the list of outbound nodes are carried upwards whena DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes ofa template, will be a superset of the outbound nodes of its last children.|
|`outputs`|[`Outputs`](#outputs)|Outputs hold parameters, artifacts, and results from a step|
|`phase`|`string`|Phase a simple, high-level summary of where the node is in its lifecycle.Can be used as a state machine.|
|`podIP`|`string`|PodIP captures the IP of the pod for daemoned steps|
|`resourcesDuration`|`Map< string , int64 >`|ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes.|
|`startedAt`|[`Time`](#time)|Time is a wrapper around time.Time which supports correctmarshaling to YAML and JSON.  Wrappers are provided for manyof the factory methods that the time package offers.|
|`storedTemplateID`|`string`|StoredTemplateID is the ID of stored template.DEPRECATED: This value is not used anymore.|
|`templateName`|`string`|TemplateName is the template name which this node corresponds to.Not applicable to virtual nodes (e.g. Retry, StepGroup)|
|`templateRef`|[`TemplateRef`](#templateref)|TemplateRef is a reference of template resource.|
|`templateScope`|`string`|TemplateScope is the template scope in which the template of this node was retrieved.|
|`type`|`string`|Type indicates type of node|
|`workflowTemplateName`|`string`|WorkflowTemplateName is the WorkflowTemplate resource name on which the resolved template of this node is retrieved.DEPRECATED: This value is not used anymore.|

## Outputs

Outputs hold parameters, artifacts, and results from a step
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifacts`|`Array<`[`Artifact`](#artifact)`>`|Artifacts holds the list of output artifacts produced by a step|
|`parameters`|`Array<`[`Parameter`](#parameter)`>`|Parameters holds the list of output parameters produced by a step|
|`result`|`string`|Result holds the result (stdout) of a script template|

## Artifact

Artifact indicates an artifact to place at a specified path
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`archive`|[`ArchiveStrategy`](#archivestrategy)|ArchiveStrategy describes how to archive files/directory when saving artifacts|
|`artifactLocation`|[`ArtifactLocation`](#artifactlocation)|ArtifactLocation describes a location for a single or multiple artifacts.It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname).It is also used to describe the location of multiple artifacts such as the archive locationof a single workflow step, which the executor will use as a default location to store its files.|
|`from`|`string`|From allows an artifact to reference an artifact from a previous step|
|`globalName`|`string`|GlobalName exports an output artifact to the global scope, making it available as'{{io.argoproj.workflow.v1alpha1.outputs.artifacts.XXXX}} and in workflow.status.outputs.artifacts|
|`mode`|`int32`|mode bits to use on this file, must be a value between 0 and 0777set when loading input artifacts.|
|`name`|`string`|name of the artifact. must be unique within a template's inputs/outputs.|
|`optional`|`boolean`|Make Artifacts optional, if Artifacts doesn't generate or exist|
|`path`|`string`|Path is the container path to the artifact|

## Parameter

Parameter indicate a passed string parameter to a service template with an optional default value
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`default`|`string`|Default is the default value to use for an input parameter if a value was not suppliedDEPRECATED: This field is not used|
|`globalName`|`string`|GlobalName exports an output parameter to the global scope, making it available as'{{io.argoproj.workflow.v1alpha1.outputs.parameters.XXXX}} and in workflow.status.outputs.parameters|
|`name`|`string`|Name is the parameter name|
|`value`|`string`|Value is the literal value to use for the parameter.If specified in the context of an input parameter, the value takes precedence over any passed values|
|`valueFrom`|[`ValueFrom`](#valuefrom)|ValueFrom describes a location in which to obtain the value to a parameter|

## Prometheus

Prometheus is a prometheus metric to be emitted
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`counter`|[`Counter`](#counter)|Counter is a Counter prometheus metric|
|`gauge`|[`Gauge`](#gauge)|Gauge is a Gauge prometheus metric|
|`help`|`string`|Help is a string that describes the metric|
|`histogram`|[`Histogram`](#histogram)|Histogram is a Histogram prometheus metric|
|`labels`|`Array<`[`MetricLabel`](#metriclabel)`>`|Labels is a list of metric labels|
|`name`|`string`|Name is the name of the metric|
|`when`|`string`|When is a conditional statement that decides when to emit the metric|

## ArtifactLocation

ArtifactLocation describes a location for a single or multiple artifacts.It is used as single artifact in the context of inputs/outputs (e.g. outputs.artifacts.artname).It is also used to describe the location of multiple artifacts such as the archive locationof a single workflow step, which the executor will use as a default location to store its files.
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`archiveLogs`|`boolean`|ArchiveLogs indicates if the container logs should be archived|
|`artifactory`|[`ArtifactoryArtifact`](#artifactoryartifact)|ArtifactoryArtifact is the location of an artifactory artifact|
|`gcs`|[`GCSArtifact`](#gcsartifact)|GCSArtifact is the location of a GCS artifact|
|`git`|[`GitArtifact`](#gitartifact)|GitArtifact is the location of an git artifact|
|`hdfs`|[`HDFSArtifact`](#hdfsartifact)|HDFSArtifact is the location of an HDFS artifact|
|`http`|[`HTTPArtifact`](#httpartifact)|HTTPArtifact allows an file served on HTTP to be placed as an input artifact in a container|
|`oss`|[`OSSArtifact`](#ossartifact)|OSSArtifact is the location of an OSS artifact|
|`raw`|[`RawArtifact`](#rawartifact)|RawArtifact allows raw string content to be placed as an artifact in a container|
|`s3`|[`S3Artifact`](#s3artifact)|S3Artifact is the location of an S3 artifact|

## DAGTemplate

DAGTemplate is a template subtype for directed acyclic graph templates
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`failFast`|`boolean`|This flag is for DAG logic. The DAG logic has a built-in "fail fast" feature to stop scheduling new steps,as soon as it detects that one of the DAG nodes is failed. Then it waits until all DAG nodes are completedbefore failing the DAG itself.The FailFast flag default is true,  if set to false, it will allow a DAG to run all branches of the DAG tocompletion (either success or failure), regardless of the failed outcomes of branches in the DAG.More info and example about this feature at https://github.com/argoproj/argo/issues/1442|
|`target`|`string`|Target are one or more names of targets to execute in a DAG|
|`tasks`|`Array<`[`DAGTask`](#dagtask)`>`|Tasks are a list of DAG tasks|

## UserContainer

UserContainer is a container specified by a user.
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`container`|[`Container`](#container)|A single application container that you want to run within a pod.|
|`mirrorVolumeMounts`|`boolean`|MirrorVolumeMounts will mount the same volumes specified in the main containerto the container (including artifacts), at the same mountPaths. This enablesdind daemon to partially see the same filesystem as the main container inorder to use features such as docker volume binding|

## Inputs

Inputs are the mechanism for passing parameters, artifacts, volumes from one template to another
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifacts`|`Array<`[`Artifact`](#artifact)`>`|Artifact are a list of artifacts passed as inputs|
|`parameters`|`Array<`[`Parameter`](#parameter)`>`|Parameters are a list of parameters passed as inputs|

## Metadata

Pod metdata
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`annotations`|`Map< string , string >`||
|`labels`|`Map< string , string >`||

## ResourceTemplate

ResourceTemplate is a template subtype to manipulate kubernetes resources
    
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
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`backoff`|[`Backoff`](#backoff)||
|`limit`|`int32`|Limit is the maximum number of attempts when retrying a container|
|`retryPolicy`|`string`|RetryPolicy is a policy of NodePhase statuses that will be retried|

## ScriptTemplate

ScriptTemplate is a template subtype to enable scripting through code steps
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`container`|[`Container`](#container)|A single application container that you want to run within a pod.|
|`source`|`string`|Source contains the source code of the script to execute|

## ParallelSteps

_No description available_

## SuspendTemplate

SuspendTemplate is a template subtype to suspend a workflow at a predetermined point in time
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`duration`|`string`|Duration is the seconds to wait before automatically resuming a template|

## TemplateRef

TemplateRef is a reference of template resource.
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`clusterscope`|`boolean`|ClusterScope indicates the referred template is cluster scoped (i.e., a ClusterWorkflowTemplate).|
|`name`|`string`|Name is the resource name of the template.|
|`runtimeResolution`|`boolean`|RuntimeResolution skips validation at creation time.By enabling this option, you can create the referred workflow template before the actual runtime.|
|`template`|`string`|Template is the name of referred template in the resource.|

## ArchiveStrategy

ArchiveStrategy describes how to archive files/directory when saving artifacts
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`none`|[`NoneStrategy`](#nonestrategy)|NoneStrategy indicates to skip tar process and upload the files or directory tree as independentfiles. Note that if the artifact is a directory, the artifact driver must support the ability tosave/load the directory appropriately.|
|`tar`|[`TarStrategy`](#tarstrategy)|TarStrategy will tar and gzip the file or directory when saving|

## ValueFrom

ValueFrom describes a location in which to obtain the value to a parameter
    
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
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`value`|`string`|Value is the value of the metric|

## Gauge

Gauge is a Gauge prometheus metric
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`realtime`|`boolean`|Realtime emits this metric in real time if applicable|
|`value`|`string`|Value is the value of the metric|

## Histogram

Histogram is a Histogram prometheus metric
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`buckets`|`Array< number >`|Buckets is a list of bucket divisors for the histogram|
|`value`|`string`|Value is the value of the metric|

## MetricLabel

MetricLabel is a single label for a prometheus metric
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`||
|`value`|`string`||

## ArtifactoryArtifact

ArtifactoryArtifact is the location of an artifactory artifact
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifactoryAuth`|[`ArtifactoryAuth`](#artifactoryauth)|ArtifactoryAuth describes the secret selectors required for authenticating to artifactory|
|`url`|`string`|URL of the artifact|

## GCSArtifact

GCSArtifact is the location of a GCS artifact
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`gCSBucket`|[`GCSBucket`](#gcsbucket)|GCSBucket contains the access information for interfacring with a GCS bucket|
|`key`|`string`|Key is the path in the bucket where the artifact resides|

## GitArtifact

GitArtifact is the location of an git artifact
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`depth`|`uint64`|Depth specifies clones/fetches should be shallow and include the givennumber of commits from the branch tip|
|`fetch`|`Array< string >`|Fetch specifies a number of refs that should be fetched before checkout|
|`insecureIgnoreHostKey`|`boolean`|InsecureIgnoreHostKey disables SSH strict host key checking during git clone|
|`passwordSecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|
|`repo`|`string`|Repo is the git repository|
|`revision`|`string`|Revision is the git commit, tag, branch to checkout|
|`sshPrivateKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|
|`usernameSecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|

## HDFSArtifact

HDFSArtifact is the location of an HDFS artifact
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`force`|`boolean`|Force copies a file forcibly even if it exists (default: false)|
|`hDFSConfig`|[`HDFSConfig`](#hdfsconfig)|HDFSConfig is configurations for HDFS|
|`path`|`string`|Path is a file path in HDFS|

## HTTPArtifact

HTTPArtifact allows an file served on HTTP to be placed as an input artifact in a container
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`url`|`string`|URL of the artifact|

## OSSArtifact

OSSArtifact is the location of an OSS artifact
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`key`|`string`|Key is the path in the bucket where the artifact resides|
|`oSSBucket`|[`OSSBucket`](#ossbucket)|OSSBucket contains the access information required for interfacing with an OSS bucket|

## RawArtifact

RawArtifact allows raw string content to be placed as an artifact in a container
    
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
|`s3Bucket`|[`S3Bucket`](#s3bucket)|S3Bucket contains the access information required for interfacing with an S3 bucket|

## DAGTask

DAGTask represents a node in the graph during DAG execution
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`arguments`|[`Arguments`](#arguments)|Arguments to a template|
|`continueOn`|[`ContinueOn`](#continueon)|ContinueOn defines if a workflow should continue even if a task or step fails/errors.It can be specified if the workflow should continue when the pod errors, fails or both.|
|`dependencies`|`Array< string >`|Dependencies are name of other targets which this depends on|
|`name`|`string`|Name is the name of the target|
|`onExit`|`string`|OnExit is a template reference which is invoked at the end of thetemplate, irrespective of the success, failure, or error of theprimary template.|
|`template`|`string`|Name of template to execute|
|`templateRef`|[`TemplateRef`](#templateref)|TemplateRef is a reference of template resource.|
|`when`|`string`|When is an expression in which the task should conditionally execute|
|`withItems`|`Array<`[`Item`](#item)`>`|WithItems expands a task into multiple parallel tasks from the items in the list|
|`withParam`|`string`|WithParam expands a task into multiple parallel tasks from the value in the parameter,which is expected to be a JSON list.|
|`withSequence`|[`Sequence`](#sequence)|Sequence expands a workflow step into numeric range|

## Backoff

_No description available_
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`duration`|`string`||
|`factor`|`int32`||
|`maxDuration`|`string`||

## NoneStrategy

NoneStrategy indicates to skip tar process and upload the files or directory tree as independentfiles. Note that if the artifact is a directory, the artifact driver must support the ability tosave/load the directory appropriately.

## TarStrategy

TarStrategy will tar and gzip the file or directory when saving
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`compressionLevel`|`int32`|CompressionLevel specifies the gzip compression level to use for the artifact.Defaults to gzip.DefaultCompression.|

## ArtifactoryAuth

ArtifactoryAuth describes the secret selectors required for authenticating to artifactory
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`passwordSecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|
|`usernameSecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|

## GCSBucket

GCSBucket contains the access information for interfacring with a GCS bucket
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`bucket`|`string`|Bucket is the name of the bucket|
|`serviceAccountKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|

## HDFSConfig

HDFSConfig is configurations for HDFS
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`addresses`|`Array< string >`|Addresses is accessible addresses of HDFS name nodes|
|`hDFSKrbConfig`|[`HDFSKrbConfig`](#hdfskrbconfig)|HDFSKrbConfig is auth configurations for Kerberos|
|`hdfsUser`|`string`|HDFSUser is the user to access HDFS file system.It is ignored if either ccache or keytab is used.|

## OSSBucket

OSSBucket contains the access information required for interfacing with an OSS bucket
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|
|`bucket`|`string`|Bucket is the name of the bucket|
|`endpoint`|`string`|Endpoint is the hostname of the bucket endpoint|
|`secretKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|

## S3Bucket

S3Bucket contains the access information required for interfacing with an S3 bucket
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`accessKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|
|`bucket`|`string`|Bucket is the name of the bucket|
|`endpoint`|`string`|Endpoint is the hostname of the bucket endpoint|
|`insecure`|`boolean`|Insecure will connect to the service with TLS|
|`region`|`string`|Region contains the optional bucket region|
|`roleARN`|`string`|RoleARN is the Amazon Resource Name (ARN) of the role to assume.|
|`secretKeySecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|
|`useSDKCreds`|`boolean`|UseSDKCreds tells the driver to figure out credentials based on sdk defaults.|

## ContinueOn

ContinueOn defines if a workflow should continue even if a task or step fails/errors.It can be specified if the workflow should continue when the pod errors, fails or both.
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`error`|`boolean`|+optional|
|`failed`|`boolean`|+optional|

## Item

_No description available_
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`boolVal`|`boolean`||
|`listVal`|`Array<`[`ItemValue`](#itemvalue)`>`||
|`mapVal`|[`ItemValue`](#itemvalue)||
|`numVal`|`string`||
|`strVal`|`string`||
|`type`|`int64`||

## Sequence

Sequence expands a workflow step into numeric range
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`count`|`string`|Count is number of elements in the sequence (default: 0). Not to be used with end|
|`end`|`string`|Number at which to end the sequence (default: 0). Not to be used with Count|
|`format`|`string`|Format is a printf format string to format the value in the sequence|
|`start`|`string`|Number at which to start the sequence (default: 0)|

## HDFSKrbConfig

HDFSKrbConfig is auth configurations for Kerberos
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`krbCCacheSecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|
|`krbConfigConfigMap`|[`ConfigMapKeySelector`](#configmapkeyselector)|Selects a key from a ConfigMap.|
|`krbKeytabSecret`|[`SecretKeySelector`](#secretkeyselector)|SecretKeySelector selects a key of a Secret.|
|`krbRealm`|`string`|KrbRealm is the Kerberos realm used with Kerberos keytabIt must be set if keytab is used.|
|`krbServicePrincipalName`|`string`|KrbServicePrincipalName is the principal name of Kerberos serviceIt must be set if either ccache or keytab is used.|
|`krbUsername`|`string`|KrbUsername is the Kerberos username used with Kerberos keytabIt must be set if keytab is used.|

## ItemValue

_No description available_
    
### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`boolVal`|`boolean`||
|`listVal`|`Array< string >`||
|`mapVal`|`Map< string , string >`||
|`numVal`|`string`||
|`strVal`|`string`||
|`type`|`int64`||
