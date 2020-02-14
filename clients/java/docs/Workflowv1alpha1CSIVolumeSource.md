

# Workflowv1alpha1CSIVolumeSource

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**driver** | **String** | Driver is the name of the CSI driver that handles this volume. Consult with your admin for the correct name as registered in the cluster. |  [optional]
**fsType** | **String** |  |  [optional]
**nodePublishSecretRef** | [**Workflowv1alpha1LocalObjectReference**](Workflowv1alpha1LocalObjectReference.md) |  |  [optional]
**readOnly** | **Boolean** |  |  [optional]
**volumeAttributes** | **Map&lt;String, String&gt;** |  |  [optional]



