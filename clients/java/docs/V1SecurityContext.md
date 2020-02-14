

# V1SecurityContext

SecurityContext holds security configuration that will be applied to a container. Some fields are present in both SecurityContext and PodSecurityContext.  When both are set, the values in SecurityContext take precedence.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**allowPrivilegeEscalation** | **Boolean** |  |  [optional]
**capabilities** | [**V1Capabilities**](V1Capabilities.md) |  |  [optional]
**privileged** | **Boolean** |  |  [optional]
**procMount** | **String** |  |  [optional]
**readOnlyRootFilesystem** | **Boolean** |  |  [optional]
**runAsGroup** | **String** |  |  [optional]
**runAsNonRoot** | **Boolean** |  |  [optional]
**runAsUser** | **String** |  |  [optional]
**seLinuxOptions** | [**V1SELinuxOptions**](V1SELinuxOptions.md) |  |  [optional]
**windowsOptions** | [**V1WindowsSecurityContextOptions**](V1WindowsSecurityContextOptions.md) |  |  [optional]



