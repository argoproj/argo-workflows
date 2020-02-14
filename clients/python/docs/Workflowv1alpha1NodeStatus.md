# Workflowv1alpha1NodeStatus

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**boundary_id** | **str** |  | [optional] 
**children** | **list[str]** |  | [optional] 
**daemoned** | **bool** |  | [optional] 
**display_name** | **str** |  | [optional] 
**finished_at** | [**V1Time**](V1Time.md) |  | [optional] 
**id** | **str** |  | [optional] 
**inputs** | [**V1alpha1Inputs**](V1alpha1Inputs.md) |  | [optional] 
**message** | **str** | A human readable message indicating details about why the node is in this condition. | [optional] 
**name** | **str** |  | [optional] 
**outbound_nodes** | **list[str]** | OutboundNodes tracks the node IDs which are considered \&quot;outbound\&quot; nodes to a template invocation. For every invocation of a template, there are nodes which we considered as \&quot;outbound\&quot;. Essentially, these are last nodes in the execution sequence to run, before the template is considered completed. These nodes are then connected as parents to a following step.  In the case of single pod steps (i.e. container, script, resource templates), this list will be nil since the pod itself is already considered the \&quot;outbound\&quot; node. In the case of DAGs, outbound nodes are the \&quot;target\&quot; tasks (tasks with no children). In the case of steps, outbound nodes are all the containers involved in the last step group. NOTE: since templates are composable, the list of outbound nodes are carried upwards when a DAG/steps template invokes another DAG/steps template. In other words, the outbound nodes of a template, will be a superset of the outbound nodes of its last children. | [optional] 
**outputs** | [**V1alpha1Outputs**](V1alpha1Outputs.md) |  | [optional] 
**phase** | **str** | Phase a simple, high-level summary of where the node is in its lifecycle. Can be used as a state machine. | [optional] 
**pod_ip** | **str** |  | [optional] 
**started_at** | [**V1Time**](V1Time.md) |  | [optional] 
**stored_template_id** | **str** | StoredTemplateID is the ID of stored template. DEPRECATED: This value is not used anymore. | [optional] 
**template_name** | **str** |  | [optional] 
**template_ref** | [**V1alpha1TemplateRef**](V1alpha1TemplateRef.md) |  | [optional] 
**template_scope** | **str** | TemplateScope is the template scope in which the template of this node was retrieved. | [optional] 
**type** | **str** |  | [optional] 
**workflow_template_name** | **str** | WorkflowTemplateName is the WorkflowTemplate resource name on which the resolved template of this node is retrieved. DEPRECATED: This value is not used anymore. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


