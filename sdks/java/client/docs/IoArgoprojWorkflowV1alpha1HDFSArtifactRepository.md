

# IoArgoprojWorkflowV1alpha1HDFSArtifactRepository

HDFSArtifactRepository defines the controller configuration for an HDFS artifact repository

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**addresses** | **List&lt;String&gt;** | Addresses is accessible addresses of HDFS name nodes |  [optional]
**dataTransferProtection** | **String** | DataTransferProtection is the protection level for HDFS data transfer. It corresponds to the dfs.data.transfer.protection configuration in HDFS. |  [optional]
**force** | **Boolean** | Force copies a file forcibly even if it exists |  [optional]
**hdfsUser** | **String** | HDFSUser is the user to access HDFS file system. It is ignored if either ccache or keytab is used. |  [optional]
**krbCCacheSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**krbConfigConfigMap** | [**io.kubernetes.client.openapi.models.V1ConfigMapKeySelector**](io.kubernetes.client.openapi.models.V1ConfigMapKeySelector.md) |  |  [optional]
**krbKeytabSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**krbRealm** | **String** | KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used. |  [optional]
**krbServicePrincipalName** | **String** | KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used. |  [optional]
**krbUsername** | **String** | KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used. |  [optional]
**pathFormat** | **String** | PathFormat is defines the format of path to store a file. Can reference workflow variables |  [optional]



