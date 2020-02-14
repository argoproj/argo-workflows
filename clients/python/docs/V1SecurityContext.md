# V1SecurityContext

SecurityContext holds security configuration that will be applied to a container. Some fields are present in both SecurityContext and PodSecurityContext.  When both are set, the values in SecurityContext take precedence.
## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**allow_privilege_escalation** | **bool** |  | [optional] 
**capabilities** | [**V1Capabilities**](V1Capabilities.md) |  | [optional] 
**privileged** | **bool** |  | [optional] 
**proc_mount** | **str** |  | [optional] 
**read_only_root_filesystem** | **bool** |  | [optional] 
**run_as_group** | **str** |  | [optional] 
**run_as_non_root** | **bool** |  | [optional] 
**run_as_user** | **str** |  | [optional] 
**se_linux_options** | [**V1SELinuxOptions**](V1SELinuxOptions.md) |  | [optional] 
**windows_options** | [**V1WindowsSecurityContextOptions**](V1WindowsSecurityContextOptions.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


