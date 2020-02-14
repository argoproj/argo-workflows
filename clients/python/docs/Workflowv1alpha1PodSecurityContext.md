# Workflowv1alpha1PodSecurityContext

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fs_group** | **str** | 1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR&#39;d with rw-rw----  If unset, the Kubelet will not modify the ownership and permissions of any volume. +optional | [optional] 
**run_as_group** | **str** |  | [optional] 
**run_as_non_root** | **bool** |  | [optional] 
**run_as_user** | **str** |  | [optional] 
**se_linux_options** | [**Workflowv1alpha1SELinuxOptions**](Workflowv1alpha1SELinuxOptions.md) |  | [optional] 
**supplemental_groups** | **list[str]** |  | [optional] 
**sysctls** | [**list[Workflowv1alpha1Sysctl]**](Workflowv1alpha1Sysctl.md) |  | [optional] 
**windows_options** | [**Workflowv1alpha1WindowsSecurityContextOptions**](Workflowv1alpha1WindowsSecurityContextOptions.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


