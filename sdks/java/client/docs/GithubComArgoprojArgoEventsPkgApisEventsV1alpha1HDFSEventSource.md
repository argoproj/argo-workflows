

# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1HDFSEventSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**addresses** | **List&lt;String&gt;** |  |  [optional]
**checkInterval** | **String** |  |  [optional]
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  |  [optional]
**hdfsUser** | **String** | HDFSUser is the user to access HDFS file system. It is ignored if either ccache or keytab is used. |  [optional]
**krbCCacheSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**krbConfigConfigMap** | [**io.kubernetes.client.openapi.models.V1ConfigMapKeySelector**](io.kubernetes.client.openapi.models.V1ConfigMapKeySelector.md) |  |  [optional]
**krbKeytabSecret** | [**io.kubernetes.client.openapi.models.V1SecretKeySelector**](io.kubernetes.client.openapi.models.V1SecretKeySelector.md) |  |  [optional]
**krbRealm** | **String** | KrbRealm is the Kerberos realm used with Kerberos keytab It must be set if keytab is used. |  [optional]
**krbServicePrincipalName** | **String** | KrbServicePrincipalName is the principal name of Kerberos service It must be set if either ccache or keytab is used. |  [optional]
**krbUsername** | **String** | KrbUsername is the Kerberos username used with Kerberos keytab It must be set if keytab is used. |  [optional]
**metadata** | **Map&lt;String, String&gt;** |  |  [optional]
**type** | **String** |  |  [optional]
**watchPathConfig** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WatchPathConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1WatchPathConfig.md) |  |  [optional]



