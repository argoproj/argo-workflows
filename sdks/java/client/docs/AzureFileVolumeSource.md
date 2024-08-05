

# AzureFileVolumeSource

AzureFile represents an Azure File Service mount on the host and bind mount to the pod.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**readOnly** | **Boolean** | readOnly defaults to false (read/write). ReadOnly here will force the ReadOnly setting in VolumeMounts. |  [optional]
**secretName** | **String** | secretName is the  name of secret that contains Azure Storage Account Name and Key | 
**shareName** | **String** | shareName is the azure share Name | 



