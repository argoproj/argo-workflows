
# Argo Types


## Workflow

Workflow is the definition of a workflow resource

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`metadata`|[`ObjectMeta`](#objectmeta)|ObjectMeta is metadata that all persisted resources must have, which includes all objectsusers must create.|
|`spec`|[`WorkflowSpec`](#workflowspec)|WorkflowSpec is the specification of a Workflow.|
|`status`|[`WorkflowStatus`](#workflowstatus)|WorkflowStatus contains overall status information about a workflow|

## ObjectMeta

ObjectMeta is metadata that all persisted resources must have, which includes all objectsusers must create.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`annotations`|`object`|Annotations is an unstructured key value map stored with a resource that may beset by external tools to store and retrieve arbitrary metadata. They are notqueryable and should be preserved when modifying objects.More info: http://kubernetes.io/docs/user-guide/annotations+optional|
|`clusterName`|`string`|The name of the cluster which the object belongs to.This is used to distinguish resources with same name and namespace in different clusters.This field is not set anywhere right now and apiserver is going to ignore it if set in create or update request.+optional|
|`creationTimestamp`|[`Time`](#time)|Time is a wrapper around time.Time which supports correctmarshaling to YAML and JSON.  Wrappers are provided for manyof the factory methods that the time package offers.|
|`deletionGracePeriodSeconds`|`int64`|Number of seconds allowed for this object to gracefully terminate beforeit will be removed from the system. Only set when deletionTimestamp is also set.May only be shortened.Read-only.+optional|
|`deletionTimestamp`|[`Time`](#time)|Time is a wrapper around time.Time which supports correctmarshaling to YAML and JSON.  Wrappers are provided for manyof the factory methods that the time package offers.|
|`finalizers`|`array`|Must be empty before the object is deleted from the registry. Each entryis an identifier for the responsible component that will remove the entryfrom the list. If the deletionTimestamp of the object is non-nil, entriesin this list can only be removed.+optional|
|`generateName`|`string`|GenerateName is an optional prefix, used by the server, to generate a uniquename ONLY IF the Name field has not been provided.If this field is used, the name returned to the client will be differentthan the name passed. This value will also be combined with a unique suffix.The provided value has the same validation rules as the Name field,and may be truncated by the length of the suffix required to make the valueunique on the server.If this field is specified and the generated name exists, the server willNOT return a 409 - instead, it will either return 201 Created or 500 with ReasonServerTimeout indicating a unique name could not be found in the time allotted, and the clientshould retry (optionally after the time indicated in the Retry-After header).Applied only if Name is not specified.More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency+optional|
|`generation`|`int64`|A sequence number representing a specific generation of the desired state.Populated by the system. Read-only.+optional|
|`labels`|`object`|Map of string keys and values that can be used to organize and categorize(scope and select) objects. May match selectors of replication controllersand services.More info: http://kubernetes.io/docs/user-guide/labels+optional|
|`managedFields`|`array`|ManagedFields maps workflow-id and version to the set of fieldsthat are managed by that io.argoproj.workflow.v1alpha1. This is mostly for internalhousekeeping, and users typically shouldn't need to set orunderstand this field. A workflow can be the user's name, acontroller's name, or the name of a specific apply path like"ci-cd". The set of fields is always in the version that theworkflow used when modifying the object.+optional|
|`name`|`string`|Name must be unique within a namespace. Is required when creating resources, althoughsome resources may allow a client to request the generation of an appropriate nameautomatically. Name is primarily intended for creation idempotence and configurationdefinition.Cannot be updated.More info: http://kubernetes.io/docs/user-guide/identifiers#names+optional|
|`namespace`|`string`|Namespace defines the space within each name must be unique. An empty namespace isequivalent to the "default" namespace, but "default" is the canonical representation.Not all objects are required to be scoped to a namespace - the value of this field forthose objects will be empty.Must be a DNS_LABEL.Cannot be updated.More info: http://kubernetes.io/docs/user-guide/namespaces+optional|
|`ownerReferences`|`array`|List of objects depended by this object. If ALL objects in the list havebeen deleted, this object will be garbage collected. If this object is managed by a controller,then an entry in this list will point to this controller, with the controller field set to true.There cannot be more than one managing controller.+optional|
|`resourceVersion`|`string`|An opaque value that represents the internal version of this object that canbe used by clients to determine when objects have changed. May be used for optimisticconcurrency, change detection, and the watch operation on a resource or set of resources.Clients must treat these values as opaque and passed unmodified back to the server.They may only be valid for a particular resource or set of resources.Populated by the system.Read-only.Value must be treated as opaque by clients and .More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency+optional|
|`selfLink`|`string`|SelfLink is a URL representing this object.Populated by the system.Read-only.DEPRECATEDKubernetes will stop propagating this field in 1.20 release and the field is plannedto be removed in 1.21 release.+optional|
|`uid`|`string`|UID is the unique in time and space value for this object. It is typically generated bythe server on successful creation of a resource and is not allowed to change on PUToperations.Populated by the system.Read-only.More info: http://kubernetes.io/docs/user-guide/identifiers#uids+optional|

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
|`hostAliases`|`array`||
|`hostNetwork`|`boolean`|Host networking requested for this workflow pod. Default to false.|
|`imagePullSecrets`|`array`|ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any imagesin pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secretscan be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet.More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod|
|`metrics`|[`Metrics`](#metrics)|Metrics are a list of metrics emitted from a Workflow/Template|
|`nodeSelector`|`object`|NodeSelector is a selector which will result in all pods of the workflowto be scheduled on the selected node(s). This is able to be overridden bya nodeSelector specified in the template.|
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
|`templates`|`array`|Templates is a list of workflow templates used in a workflow|
|`tolerations`|`array`|Tolerations to apply to workflow pods.|
|`ttlSecondsAfterFinished`|`int32`|TTLSecondsAfterFinished limits the lifetime of a Workflow that has finished execution(Succeeded, Failed, Error). If this field is set, once the Workflow finishes, it will bedeleted after ttlSecondsAfterFinished expires. If this field is unset,ttlSecondsAfterFinished will not expire. If this field is set to zero,ttlSecondsAfterFinished expires immediately after the Workflow finishes.DEPRECATED: Use TTLStrategy.SecondsAfterCompletion instead.|
|`ttlStrategy`|[`TTLStrategy`](#ttlstrategy)|TTLStrategy is the strategy for the time to live depending on if the workflow succeded or failed|
|`volumeClaimTemplates`|`array`|VolumeClaimTemplates is a list of claims that containers are allowed to reference.The Workflow controller will create the claims at the beginning of the workflowand delete the claims upon completion of the workflow|
|`volumes`|`array`|Volumes is a list of volumes that can be mounted by containers in a io.argoproj.workflow.v1alpha1.|

## WorkflowStatus

WorkflowStatus contains overall status information about a workflow

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`compressedNodes`|`string`|Compressed and base64 decoded Nodes map|
|`conditions`|`array`|Conditions is a list of conditions the Workflow may have|
|`finishedAt`|[`Time`](#time)|Time is a wrapper around time.Time which supports correctmarshaling to YAML and JSON.  Wrappers are provided for manyof the factory methods that the time package offers.|
|`message`|`string`|A human readable message indicating details about why the workflow is in this condition.|
|`nodes`|`object`|Nodes is a mapping between a node ID and the node's status.|
|`offloadNodeStatusVersion`|`string`|Whether on not node status has been offloaded to a database. If exists, then Nodes and CompressedNodes will be empty.This will actually be populated with a hash of the offloaded data.|
|`outputs`|[`Outputs`](#outputs)|Outputs hold parameters, artifacts, and results from a step|
|`persistentVolumeClaims`|`array`|PersistentVolumeClaims tracks all PVCs that were created as part of the io.argoproj.workflow.v1alpha1.The contents of this list are drained at the end of the workflow.|
|`phase`|`string`|Phase a simple, high-level summary of where the workflow is in its lifecycle.|
|`resourcesDuration`|`object`|ResourcesDuration is the total for the workflow|
|`startedAt`|[`Time`](#time)|Time is a wrapper around time.Time which supports correctmarshaling to YAML and JSON.  Wrappers are provided for manyof the factory methods that the time package offers.|
|`storedTemplates`|`object`|StoredTemplates is a mapping between a template ref and the node's status.|

## Time

Time is a wrapper around time.Time which supports correctmarshaling to YAML and JSON.  Wrappers are provided for manyof the factory methods that the time package offers.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nanos`|`int32`|Non-negative fractions of a second at nanosecond resolution. Negativesecond values with fractions must still have non-negative nanos valuesthat count forward in time. Must be from 0 to 999,999,999inclusive. This field may be limited in precision depending on context.|
|`seconds`|`int64`|Represents seconds of UTC time since Unix epoch1970-01-01T00:00:00Z. Must be from 0001-01-01T00:00:00Z to9999-12-31T23:59:59Z inclusive.|

## Affinity

Affinity is a group of affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nodeAffinity`|[`NodeAffinity`](#nodeaffinity)|Node affinity is a group of node affinity scheduling rules.|
|`podAffinity`|[`PodAffinity`](#podaffinity)|Pod affinity is a group of inter pod affinity scheduling rules.|
|`podAntiAffinity`|[`PodAntiAffinity`](#podantiaffinity)|Pod anti affinity is a group of inter pod anti affinity scheduling rules.|

## Arguments

Arguments to a template

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifacts`|`array`|Artifacts is the list of artifacts to pass to the template or workflow|
|`parameters`|`array`|Parameters is the list of parameters to pass to the template or workflow|

## ArtifactRepositoryRef



### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`configMap`|`string`||
|`key`|`string`||

## PodDNSConfig

PodDNSConfig defines the DNS parameters of a pod in addition tothose generated from DNSPolicy.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nameservers`|`array`|A list of DNS name server IP addresses.This will be appended to the base nameservers generated from DNSPolicy.Duplicated nameservers will be removed.+optional|
|`options`|`array`|A list of DNS resolver options.This will be merged with the base options generated from DNSPolicy.Duplicated entries will be removed. Resolution options given in Optionswill override those that appear in the base DNSPolicy.+optional|
|`searches`|`array`|A list of DNS search domains for host-name lookup.This will be appended to the base search paths generated from DNSPolicy.Duplicated search paths will be removed.+optional|

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
|`prometheus`|`array`|Prometheus is a list of prometheus metrics to be emitted|

## PodDisruptionBudgetSpec

PodDisruptionBudgetSpec is a description of a PodDisruptionBudget.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`maxUnavailable`|[`IntOrString`](#intorstring)||
|`minAvailable`|[`IntOrString`](#intorstring)||
|`selector`|[`LabelSelector`](#labelselector)|A label selector is a label query over a set of resources. The result of matchLabels andmatchExpressions are ANDed. An empty label selector matches all objects. A nulllabel selector matches no objects.|

## PodGC

PodGC describes how to delete completed pods as they complete

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`strategy`|`string`||

## PodSecurityContext

PodSecurityContext holds pod-level security attributes and common container settings.Some fields are also present in container.securityContext.  Field values ofcontainer.securityContext take precedence over field values of PodSecurityContext.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`fsGroup`|`int64`|1. The owning GID will be the FSGroup2. The setgid bit is set (new files created in the volume will be owned by FSGroup)3. The permission bits are OR'd with rw-rw----If unset, the Kubelet will not modify the ownership and permissions of any volume.+optional|
|`runAsGroup`|`int64`|The GID to run the entrypoint of the container process.Uses runtime default if unset.May also be set in SecurityContext.  If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedencefor that container.+optional|
|`runAsNonRoot`|`boolean`|Indicates that the container must run as a non-root user.If true, the Kubelet will validate the image at runtime to ensure that itdoes not run as UID 0 (root) and fail to start the container if it does.If unset or false, no such validation will be performed.May also be set in SecurityContext.  If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedence.+optional|
|`runAsUser`|`int64`|The UID to run the entrypoint of the container process.Defaults to user specified in image metadata if unspecified.May also be set in SecurityContext.  If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedencefor that container.+optional|
|`seLinuxOptions`|[`SELinuxOptions`](#selinuxoptions)|SELinuxOptions are the labels to be applied to the container|
|`supplementalGroups`|`array`|A list of groups applied to the first process run in each container, in additionto the container's primary GID.  If unspecified, no groups will be added toany container.+optional|
|`sysctls`|`array`|Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupportedsysctls (by the container runtime) might fail to launch.+optional|
|`windowsOptions`|[`WindowsSecurityContextOptions`](#windowssecuritycontextoptions)|WindowsSecurityContextOptions contain Windows-specific options and credentials.|

## TTLStrategy

TTLStrategy is the strategy for the time to live depending on if the workflow succeded or failed

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`secondsAfterCompletion`|`int32`||
|`secondsAfterFailure`|`int32`||
|`secondsAfterSuccess`|`int32`||

## Outputs

Outputs hold parameters, artifacts, and results from a step

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`artifacts`|`array`|Artifacts holds the list of output artifacts produced by a step|
|`parameters`|`array`|Parameters holds the list of output parameters produced by a step|
|`result`|`string`|Result holds the result (stdout) of a script template|

## NodeAffinity

Node affinity is a group of node affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preferredDuringSchedulingIgnoredDuringExecution`|`array`|The scheduler will prefer to schedule pods to nodes that satisfythe affinity expressions specified by this field, but it may choosea node that violates one or more of the expressions. The node that ismost preferred is the one with the greatest sum of weights, i.e.for each node that meets all of the scheduling requirements (resourcerequest, requiredDuringScheduling affinity expressions, etc.),compute a sum by iterating through the elements of this field and adding"weight" to the sum if the node matches the corresponding matchExpressions; thenode(s) with the highest sum are the most preferred.+optional|
|`requiredDuringSchedulingIgnoredDuringExecution`|[`NodeSelector`](#nodeselector)|A node selector represents the union of the results of one or more label queriesover a set of nodes; that is, it represents the OR of the selectors representedby the node selector terms.|

## PodAffinity

Pod affinity is a group of inter pod affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preferredDuringSchedulingIgnoredDuringExecution`|`array`|The scheduler will prefer to schedule pods to nodes that satisfythe affinity expressions specified by this field, but it may choosea node that violates one or more of the expressions. The node that ismost preferred is the one with the greatest sum of weights, i.e.for each node that meets all of the scheduling requirements (resourcerequest, requiredDuringScheduling affinity expressions, etc.),compute a sum by iterating through the elements of this field and adding"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; thenode(s) with the highest sum are the most preferred.+optional|
|`requiredDuringSchedulingIgnoredDuringExecution`|`array`|If the affinity requirements specified by this field are not met atscheduling time, the pod will not be scheduled onto the node.If the affinity requirements specified by this field cease to be metat some point during pod execution (e.g. due to a pod label update), thesystem may or may not try to eventually evict the pod from its node.When there are multiple elements, the lists of nodes corresponding to eachpodAffinityTerm are intersected, i.e. all terms must be satisfied.+optional|

## PodAntiAffinity

Pod anti affinity is a group of inter pod anti affinity scheduling rules.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`preferredDuringSchedulingIgnoredDuringExecution`|`array`|The scheduler will prefer to schedule pods to nodes that satisfythe anti-affinity expressions specified by this field, but it may choosea node that violates one or more of the expressions. The node that ismost preferred is the one with the greatest sum of weights, i.e.for each node that meets all of the scheduling requirements (resourcerequest, requiredDuringScheduling anti-affinity expressions, etc.),compute a sum by iterating through the elements of this field and adding"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; thenode(s) with the highest sum are the most preferred.+optional|
|`requiredDuringSchedulingIgnoredDuringExecution`|`array`|If the anti-affinity requirements specified by this field are not met atscheduling time, the pod will not be scheduled onto the node.If the anti-affinity requirements specified by this field cease to be metat some point during pod execution (e.g. due to a pod label update), thesystem may or may not try to eventually evict the pod from its node.When there are multiple elements, the lists of nodes corresponding to eachpodAffinityTerm are intersected, i.e. all terms must be satisfied.+optional|

## IntOrString



### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`intVal`|`int32`||
|`strVal`|`string`||
|`type`|`int64`||

## LabelSelector

A label selector is a label query over a set of resources. The result of matchLabels andmatchExpressions are ANDed. An empty label selector matches all objects. A nulllabel selector matches no objects.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`matchExpressions`|`array`|matchExpressions is a list of label selector requirements. The requirements are ANDed.+optional|
|`matchLabels`|`object`|matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabelsmap is equivalent to an element of matchExpressions, whose key field is "key", theoperator is "In", and the values array contains only "value". The requirements are ANDed.+optional|

## SELinuxOptions

SELinuxOptions are the labels to be applied to the container

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`level`|`string`|Level is SELinux level label that applies to the container.+optional|
|`role`|`string`|Role is a SELinux role label that applies to the container.+optional|
|`type`|`string`|Type is a SELinux type label that applies to the container.+optional|
|`user`|`string`|User is a SELinux user label that applies to the container.+optional|

## WindowsSecurityContextOptions

WindowsSecurityContextOptions contain Windows-specific options and credentials.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`gmsaCredentialSpec`|`string`|GMSACredentialSpec is where the GMSA admission webhook(https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of theGMSA credential spec named by the GMSACredentialSpecName field.This field is alpha-level and is only honored by servers that enable the WindowsGMSA feature flag.+optional|
|`gmsaCredentialSpecName`|`string`|GMSACredentialSpecName is the name of the GMSA credential spec to use.This field is alpha-level and is only honored by servers that enable the WindowsGMSA feature flag.+optional|
|`runAsUserName`|`string`|The UserName in Windows to run the entrypoint of the container process.Defaults to the user specified in image metadata if unspecified.May also be set in PodSecurityContext. If set in both SecurityContext andPodSecurityContext, the value specified in SecurityContext takes precedence.This field is alpha-level and it is only honored by servers that enable the WindowsRunAsUserName feature flag.+optional|

## NodeSelector

A node selector represents the union of the results of one or more label queriesover a set of nodes; that is, it represents the OR of the selectors representedby the node selector terms.

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|
|`nodeSelectorTerms`|`array`|Required. A list of node selector terms. The terms are ORed.|
