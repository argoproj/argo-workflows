

# V1alpha1PodSecurityContext

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fsGroup** | **String** | 1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR&#39;d with rw-rw----  If unset, the Kubelet will not modify the ownership and permissions of any volume. +optional |  [optional]
**runAsGroup** | **String** |  |  [optional]
**runAsNonRoot** | **Boolean** |  |  [optional]
**runAsUser** | **String** |  |  [optional]
**seLinuxOptions** | [**V1alpha1SELinuxOptions**](V1alpha1SELinuxOptions.md) |  |  [optional]
**supplementalGroups** | **List&lt;String&gt;** |  |  [optional]
**sysctls** | [**List&lt;V1alpha1Sysctl&gt;**](V1alpha1Sysctl.md) |  |  [optional]
**windowsOptions** | [**V1alpha1WindowsSecurityContextOptions**](V1alpha1WindowsSecurityContextOptions.md) |  |  [optional]



