# \InfoServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**collect_event**](InfoServiceApi.md#collect_event) | **POST** /api/v1/tracking/event | 
[**get_info**](InfoServiceApi.md#get_info) | **GET** /api/v1/info | 
[**get_user_info**](InfoServiceApi.md#get_user_info) | **GET** /api/v1/userinfo | 
[**get_version**](InfoServiceApi.md#get_version) | **GET** /api/v1/version | 



## collect_event

> serde_json::Value collect_event(body)


### Parameters


Name | Type | Description  | Required | Notes
------------- | ------------- | ------------- | ------------- | -------------
**body** | [**IoArgoprojWorkflowV1alpha1CollectEventRequest**](IoArgoprojWorkflowV1alpha1CollectEventRequest.md) |  | [required] |

### Return type

[**serde_json::Value**](serde_json::Value.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## get_info

> crate::models::IoArgoprojWorkflowV1alpha1InfoResponse get_info()


### Parameters

This endpoint does not need any parameter.

### Return type

[**crate::models::IoArgoprojWorkflowV1alpha1InfoResponse**](io.argoproj.workflow.v1alpha1.InfoResponse.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## get_user_info

> crate::models::IoArgoprojWorkflowV1alpha1GetUserInfoResponse get_user_info()


### Parameters

This endpoint does not need any parameter.

### Return type

[**crate::models::IoArgoprojWorkflowV1alpha1GetUserInfoResponse**](io.argoproj.workflow.v1alpha1.GetUserInfoResponse.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)


## get_version

> crate::models::IoArgoprojWorkflowV1alpha1Version get_version()


### Parameters

This endpoint does not need any parameter.

### Return type

[**crate::models::IoArgoprojWorkflowV1alpha1Version**](io.argoproj.workflow.v1alpha1.Version.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

