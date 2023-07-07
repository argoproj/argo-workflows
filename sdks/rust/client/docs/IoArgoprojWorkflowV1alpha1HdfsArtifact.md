# IoArgoprojWorkflowV1alpha1HdfsArtifact

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**addresses** | Option<**Vec<String>**> | Addresses is accessible addresses of HDFS name nodes | [optional]
**force** | Option<**bool**> | Force copies a file forcibly even if it exists | [optional]
**hdfs_user** | Option<**String**> | HDFSUser is the user to access HDFS file system. It is ignored if either ccache or keytab is used. | [optional]
**krb_c_cache_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**krb_config_config_map** | Option<[**crate::models::ConfigMapKeySelector**](ConfigMapKeySelector.md)> |  | [optional]
**krb_keytab_secret** | Option<[**crate::models::SecretKeySelector**](SecretKeySelector.md)> |  | [optional]
**krb_realm** | Option<**String**> | KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used. | [optional]
**krb_service_principal_name** | Option<**String**> | KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used. | [optional]
**krb_username** | Option<**String**> | KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used. | [optional]
**path** | **String** | Path is a file path in HDFS | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


