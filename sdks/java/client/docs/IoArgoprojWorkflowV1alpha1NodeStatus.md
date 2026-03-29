

# IoArgoprojWorkflowV1alpha1NodeStatus

NodeStatus contains status information about an individual node in the workflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**boundaryID** | **String** | BoundaryID indicates the node ID of the associated template root node in which this node belongs to |  [optional]
**children** | **List&lt;String&gt;** | Children is a list of child node IDs |  [optional]
**daemoned** | **Boolean** | Daemoned tracks whether or not this node was daemoned and need to be terminated |  [optional]
**displayName** | **String** | DisplayName is a human readable representation of the node. Unique within a template boundary |  [optional]
**estimatedDuration** | **Integer** | EstimatedDuration in seconds. |  [optional]
**failedPodRestarts** | **Integer** | FailedPodRestarts tracks the number of times the pod for this node was restarted due to infrastructure failures before the main container started. |  [optional]
**finishedAt** | **java.time.Instant** |  |  [optional]
**hostNodeName** | **String** | HostNodeName name of the Kubernetes node on which the Pod is running, if applicable |  [optional]
**id** | **String** | ID is a unique identifier of a node within the worklow It is implemented as a hash of the node name, which makes the ID deterministic | 
**inputs** | [**IoArgoprojWorkflowV1alpha1Inputs**](IoArgoprojWorkflowV1alpha1Inputs.md) |  |  [optional]
**memoizationStatus** | [**IoArgoprojWorkflowV1alpha1MemoizationStatus**](IoArgoprojWorkflowV1alpha1MemoizationStatus.md) |  |  [optional]
**message** | **String** | A human readable message indicating details about why the node is in this condition. |  [optional]
**name** | **String** | Name is unique name in the node tree used to generate the node ID | 
**nodeFlag** | [**IoArgoprojWorkflowV1alpha1NodeFlag**](IoArgoprojWorkflowV1alpha1NodeFlag.md) |  |  [optional]
**outboundNodes** | **List&lt;String&gt;** | OutboundNodes tracks the node IDs which are considered \&quot;outbound\&quot; nodes to a template invocation. For every invocation of a template, there are nodes which we considered as \&quot;outbound\&quot;. Essentially, these are last nodes in the execution sequence to run, before the template is considered completed. These nodes are then connected as parents to a following step.  In the case of single pod steps (i.e. container, script, resource templates), this list will be nil since the pod itself is already considered the \&quot;outbound\&quot; node. In the case of DAGs, outbound nodes are the \&quot;target\&quot; tasks (tasks with no children). In the case of steps, outbound nodes are all the containers involved in the last step group. NOTE: since templates are composable, the list of outbound nodes are carried upwards when a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of a template, will be a superset of the outbound nodes of its last children. |  [optional]
**outputs** | [**IoArgoprojWorkflowV1alpha1Outputs**](IoArgoprojWorkflowV1alpha1Outputs.md) |  |  [optional]
**phase** | **String** | Phase a simple, high-level summary of where the node is in its lifecycle. Can be used as a state machine. Will be one of these values \&quot;Pending\&quot;, \&quot;Running\&quot; before the node is completed, or \&quot;Succeeded\&quot;, \&quot;Skipped\&quot;, \&quot;Failed\&quot;, \&quot;Error\&quot;, or \&quot;Omitted\&quot; as a final state. |  [optional]
**podIP** | **String** | PodIP captures the IP of the pod for daemoned steps |  [optional]
**progress** | **String** | Progress to completion |  [optional]
**resourcesDuration** | **Map&lt;String, Long&gt;** | ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes. |  [optional]
**restartingPodUID** | **String** | RestartingPodUID tracks the UID of the pod that is currently being restarted. This prevents duplicate restart attempts when the controller processes the same failed pod multiple times. Cleared when the replacement pod starts running. |  [optional]
**startedAt** | **java.time.Instant** |  |  [optional]
**synchronizationStatus** | [**IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus**](IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus.md) |  |  [optional]
**taskResultSynced** | **Boolean** | TaskResultSynced is used to determine if the node&#39;s output has been received |  [optional]
**templateName** | **String** | TemplateName is the template name which this node corresponds to. Not applicable to virtual nodes (e.g. Retry, StepGroup) |  [optional]
**templateRef** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  |  [optional]
**templateScope** | **String** | TemplateScope is the template scope in which the template of this node was retrieved. |  [optional]
**type** | **String** | Type indicates type of node | 



