# Workflowv1alpha1SecurityContext

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**allow_privilege_escalation** | **bool** |  | [optional] 
**capabilities** | [**Workflowv1alpha1Capabilities**](Workflowv1alpha1Capabilities.md) |  | [optional] 
**privileged** | **bool** |  | [optional] 
**proc_mount** | **str** |  | [optional] 
**read_only_root_filesystem** | **bool** |  | [optional] 
**run_as_group** | **str** |  | [optional] 
**run_as_non_root** | **bool** |  | [optional] 
**run_as_user** | **str** |  | [optional] 
**se_linux_options** | [**Workflowv1alpha1SELinuxOptions**](Workflowv1alpha1SELinuxOptions.md) |  | [optional] 
**windows_options** | [**Workflowv1alpha1WindowsSecurityContextOptions**](Workflowv1alpha1WindowsSecurityContextOptions.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


