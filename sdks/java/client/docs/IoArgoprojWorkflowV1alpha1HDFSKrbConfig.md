

# IoArgoprojWorkflowV1alpha1HDFSKrbConfig


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**krbCCacheSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**krbConfigConfigMap** | [**io.kubernetes.client.openapi.models.V1ConfigMapKeySelector**](io.kubernetes.client.openapi.models.V1ConfigMapKeySelector.md) |  |  [optional]
**krbKeytabSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**krbRealm** | **String** | KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used. |  [optional]
**krbServicePrincipalName** | **String** | KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used. |  [optional]
**krbUsername** | **String** | KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used. |  [optional]



