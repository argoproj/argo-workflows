

# Workflowv1alpha1PodSecurityContext

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**fsGroup** | **String** | 1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR&#39;d with rw-rw----  If unset, the Kubelet will not modify the ownership and permissions of any volume. +optional |  [optional]
**runAsGroup** | **String** |  |  [optional]
**runAsNonRoot** | **Boolean** |  |  [optional]
**runAsUser** | **String** |  |  [optional]
**seLinuxOptions** | [**Workflowv1alpha1SELinuxOptions**](Workflowv1alpha1SELinuxOptions.md) |  |  [optional]
**supplementalGroups** | **List&lt;String&gt;** |  |  [optional]
**sysctls** | [**List&lt;Workflowv1alpha1Sysctl&gt;**](Workflowv1alpha1Sysctl.md) |  |  [optional]
**windowsOptions** | [**Workflowv1alpha1WindowsSecurityContextOptions**](Workflowv1alpha1WindowsSecurityContextOptions.md) |  |  [optional]



