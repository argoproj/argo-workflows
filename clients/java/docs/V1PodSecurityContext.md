

# V1PodSecurityContext

PodSecurityContext holds pod-level security attributes and common container settings. Some fields are also present in container.securityContext.  Field values of container.securityContext take precedence over field values of PodSecurityContext.
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fsGroup** | **String** | 1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR&#39;d with rw-rw----  If unset, the Kubelet will not modify the ownership and permissions of any volume. +optional |  [optional]
**runAsGroup** | **String** |  |  [optional]
**runAsNonRoot** | **Boolean** |  |  [optional]
**runAsUser** | **String** |  |  [optional]
**seLinuxOptions** | [**V1SELinuxOptions**](V1SELinuxOptions.md) |  |  [optional]
**supplementalGroups** | **List&lt;String&gt;** |  |  [optional]
**sysctls** | [**List&lt;V1Sysctl&gt;**](V1Sysctl.md) |  |  [optional]
**windowsOptions** | [**V1WindowsSecurityContextOptions**](V1WindowsSecurityContextOptions.md) |  |  [optional]



