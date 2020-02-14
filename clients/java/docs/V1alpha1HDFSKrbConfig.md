

# V1alpha1HDFSKrbConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**krbCCacheSecret** | [**V1SecretKeySelector**](V1SecretKeySelector.md) |  |  [optional]
**krbConfigConfigMap** | [**V1ConfigMapKeySelector**](V1ConfigMapKeySelector.md) |  |  [optional]
**krbKeytabSecret** | [**V1SecretKeySelector**](V1SecretKeySelector.md) |  |  [optional]
**krbRealm** | **String** | KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used. |  [optional]
**krbServicePrincipalName** | **String** | KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used. |  [optional]
**krbUsername** | **String** | KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used. |  [optional]



