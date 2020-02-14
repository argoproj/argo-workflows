# V1alpha1HDFSKrbConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**krb_c_cache_secret** | [**V1SecretKeySelector**](V1SecretKeySelector.md) |  | [optional] 
**krb_config_config_map** | [**V1ConfigMapKeySelector**](V1ConfigMapKeySelector.md) |  | [optional] 
**krb_keytab_secret** | [**V1SecretKeySelector**](V1SecretKeySelector.md) |  | [optional] 
**krb_realm** | **str** | KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used. | [optional] 
**krb_service_principal_name** | **str** | KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used. | [optional] 
**krb_username** | **str** | KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


