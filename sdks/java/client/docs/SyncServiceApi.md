# SyncServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**syncServiceCreateSyncLimit**](SyncServiceApi.md#syncServiceCreateSyncLimit) | **POST** /api/v1/sync/{namespace} | 
[**syncServiceDeleteSyncLimit**](SyncServiceApi.md#syncServiceDeleteSyncLimit) | **DELETE** /api/v1/sync/{namespace}/{name} | 
[**syncServiceGetSyncLimit**](SyncServiceApi.md#syncServiceGetSyncLimit) | **GET** /api/v1/sync/{namespace}/{name} | 
[**syncServiceUpdateSyncLimit**](SyncServiceApi.md#syncServiceUpdateSyncLimit) | **PUT** /api/v1/sync/{namespace}/{name} | 


<a name="syncServiceCreateSyncLimit"></a>
# **syncServiceCreateSyncLimit**
> SyncSyncLimitResponse syncServiceCreateSyncLimit(namespace, body)



### Example
```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.auth.*;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.SyncServiceApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:2746");
    
    // Configure API key authorization: BearerToken
    ApiKeyAuth BearerToken = (ApiKeyAuth) defaultClient.getAuthentication("BearerToken");
    BearerToken.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //BearerToken.setApiKeyPrefix("Token");

    SyncServiceApi apiInstance = new SyncServiceApi(defaultClient);
    String namespace = "namespace_example"; // String | 
    SyncCreateSyncLimitRequest body = new SyncCreateSyncLimitRequest(); // SyncCreateSyncLimitRequest | 
    try {
      SyncSyncLimitResponse result = apiInstance.syncServiceCreateSyncLimit(namespace, body);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling SyncServiceApi#syncServiceCreateSyncLimit");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **String**|  |
 **body** | [**SyncCreateSyncLimitRequest**](SyncCreateSyncLimitRequest.md)|  |

### Return type

[**SyncSyncLimitResponse**](SyncSyncLimitResponse.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

<a name="syncServiceDeleteSyncLimit"></a>
# **syncServiceDeleteSyncLimit**
> Object syncServiceDeleteSyncLimit(namespace, name, type, key)



### Example
```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.auth.*;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.SyncServiceApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:2746");
    
    // Configure API key authorization: BearerToken
    ApiKeyAuth BearerToken = (ApiKeyAuth) defaultClient.getAuthentication("BearerToken");
    BearerToken.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //BearerToken.setApiKeyPrefix("Token");

    SyncServiceApi apiInstance = new SyncServiceApi(defaultClient);
    String namespace = "namespace_example"; // String | 
    String name = "name_example"; // String | 
    String type = "CONFIG_MAP"; // String | 
    String key = "key_example"; // String | 
    try {
      Object result = apiInstance.syncServiceDeleteSyncLimit(namespace, name, type, key);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling SyncServiceApi#syncServiceDeleteSyncLimit");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **String**|  |
 **name** | **String**|  |
 **type** | **String**|  | [optional] [default to CONFIG_MAP] [enum: CONFIG_MAP, DATABASE]
 **key** | **String**|  | [optional]

### Return type

**Object**

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

<a name="syncServiceGetSyncLimit"></a>
# **syncServiceGetSyncLimit**
> SyncSyncLimitResponse syncServiceGetSyncLimit(namespace, name, type, key)



### Example
```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.auth.*;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.SyncServiceApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:2746");
    
    // Configure API key authorization: BearerToken
    ApiKeyAuth BearerToken = (ApiKeyAuth) defaultClient.getAuthentication("BearerToken");
    BearerToken.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //BearerToken.setApiKeyPrefix("Token");

    SyncServiceApi apiInstance = new SyncServiceApi(defaultClient);
    String namespace = "namespace_example"; // String | 
    String name = "name_example"; // String | 
    String type = "CONFIG_MAP"; // String | 
    String key = "key_example"; // String | 
    try {
      SyncSyncLimitResponse result = apiInstance.syncServiceGetSyncLimit(namespace, name, type, key);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling SyncServiceApi#syncServiceGetSyncLimit");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **String**|  |
 **name** | **String**|  |
 **type** | **String**|  | [optional] [default to CONFIG_MAP] [enum: CONFIG_MAP, DATABASE]
 **key** | **String**|  | [optional]

### Return type

[**SyncSyncLimitResponse**](SyncSyncLimitResponse.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

<a name="syncServiceUpdateSyncLimit"></a>
# **syncServiceUpdateSyncLimit**
> SyncSyncLimitResponse syncServiceUpdateSyncLimit(namespace, name)



### Example
```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.auth.*;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.SyncServiceApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:2746");
    
    // Configure API key authorization: BearerToken
    ApiKeyAuth BearerToken = (ApiKeyAuth) defaultClient.getAuthentication("BearerToken");
    BearerToken.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //BearerToken.setApiKeyPrefix("Token");

    SyncServiceApi apiInstance = new SyncServiceApi(defaultClient);
    String namespace = "namespace_example"; // String | 
    String name = "name_example"; // String | 
    try {
      SyncSyncLimitResponse result = apiInstance.syncServiceUpdateSyncLimit(namespace, name);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling SyncServiceApi#syncServiceUpdateSyncLimit");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **String**|  |
 **name** | **String**|  |

### Return type

[**SyncSyncLimitResponse**](SyncSyncLimitResponse.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |
**0** | An unexpected error response. |  -  |

