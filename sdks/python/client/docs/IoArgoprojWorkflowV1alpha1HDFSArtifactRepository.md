# IoArgoprojWorkflowV1alpha1HDFSArtifactRepository

HDFSArtifactRepository defines the controller configuration for an HDFS artifact repository

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**addresses** | **[str]** | Addresses is accessible addresses of HDFS name nodes | [optional] 
**force** | **bool** | Force copies a file forcibly even if it exists | [optional] 
**hdfs_user** | **str** | HDFSUser is the user to access HDFS file system. It is ignored if either ccache or keytab is used. | [optional] 
**krb_c_cache_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**krb_config_config_map** | [**ConfigMapKeySelector**](ConfigMapKeySelector.md) |  | [optional] 
**krb_keytab_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**krb_realm** | **str** | KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used. | [optional] 
**krb_service_principal_name** | **str** | KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used. | [optional] 
**krb_username** | **str** | KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used. | [optional] 
**path_format** | **str** | PathFormat is defines the format of path to store a file. Can reference workflow variables | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


