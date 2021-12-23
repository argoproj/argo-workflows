

# IoArgoprojWorkflowV1alpha1NodeStatus


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**boundaryID** | **String** |  |  [optional]
**children** | **List&lt;String&gt;** |  |  [optional]
**daemoned** | **Boolean** |  |  [optional]
**displayName** | **String** |  |  [optional]
**estimatedDuration** | **String** | EstimatedDuration in seconds. |  [optional]
**finishedAt** | **java.time.Instant** |  |  [optional]
**hostNodeName** | **String** |  |  [optional]
**id** | **String** |  |  [optional]
**inputs** | [**IoArgoprojWorkflowV1alpha1Inputs**](IoArgoprojWorkflowV1alpha1Inputs.md) |  |  [optional]
**memoizationStatus** | [**IoArgoprojWorkflowV1alpha1MemoizationStatus**](IoArgoprojWorkflowV1alpha1MemoizationStatus.md) |  |  [optional]
**message** | **String** | A human readable message indicating details about why the node is in this condition. |  [optional]
**name** | **String** |  |  [optional]
**outboundNodes** | **List&lt;String&gt;** | OutboundNodes tracks the node IDs which are considered \&quot;outbound\&quot; nodes to a template invocation. For every invocation of a template, there are nodes which we considered as \&quot;outbound\&quot;. Essentially, these are last nodes in the execution sequence to run, before the template is considered completed. These nodes are then connected as parents to a following step.  In the case of single pod steps (i.e. container, script, resource templates), this list will be nil since the pod itself is already considered the \&quot;outbound\&quot; node. In the case of DAGs, outbound nodes are the \&quot;target\&quot; tasks (tasks with no children). In the case of steps, outbound nodes are all the containers involved in the last step group. NOTE: since templates are composable, the list of outbound nodes are carried upwards when a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of a template, will be a superset of the outbound nodes of its last children. |  [optional]
**outputs** | [**IoArgoprojWorkflowV1alpha1Outputs**](IoArgoprojWorkflowV1alpha1Outputs.md) |  |  [optional]
**phase** | **String** | Phase a simple, high-level summary of where the node is in its lifecycle. Can be used as a state machine. |  [optional]
**podIP** | **String** |  |  [optional]
**progress** | **String** |  |  [optional]
**resourcesDuration** | **Map&lt;String, String&gt;** | ResourcesDuration is indicative, but not accurate, resource duration. This is populated when the nodes completes. |  [optional]
**startedAt** | **java.time.Instant** |  |  [optional]
**synchronizationStatus** | [**IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus**](IoArgoprojWorkflowV1alpha1NodeSynchronizationStatus.md) |  |  [optional]
**templateName** | **String** |  |  [optional]
**templateRef** | [**IoArgoprojWorkflowV1alpha1TemplateRef**](IoArgoprojWorkflowV1alpha1TemplateRef.md) |  |  [optional]
**templateScope** | **String** | TemplateScope is the template scope in which the template of this node was retrieved. |  [optional]
**type** | **String** |  |  [optional]



