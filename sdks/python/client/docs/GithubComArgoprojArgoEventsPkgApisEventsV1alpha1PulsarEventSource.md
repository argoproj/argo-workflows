# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1PulsarEventSource


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**auth_athenz_params** | **{str: (str,)}** |  | [optional] 
**auth_athenz_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**auth_token_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**connection_backoff** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1Backoff.md) |  | [optional] 
**filter** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventSourceFilter.md) |  | [optional] 
**json_body** | **bool** |  | [optional] 
**metadata** | **{str: (str,)}** |  | [optional] 
**tls** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TLSConfig.md) |  | [optional] 
**tls_allow_insecure_connection** | **bool** |  | [optional] 
**tls_trust_certs_secret** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**tls_validate_hostname** | **bool** |  | [optional] 
**topics** | **[str]** |  | [optional] 
**type** | **str** |  | [optional] 
**url** | **str** |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


