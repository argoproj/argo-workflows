# ArtifactServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**artifactServiceGetInputArtifact**](ArtifactServiceApi.md#artifactServiceGetInputArtifact) | **GET** /input-artifacts/{namespace}/{name}/{nodeId}/{artifactName} | Get an input artifact.
[**artifactServiceGetInputArtifactByUID**](ArtifactServiceApi.md#artifactServiceGetInputArtifactByUID) | **GET** /input-artifacts-by-uid/{uid}/{nodeId}/{artifactName} | Get an input artifact by UID.
[**artifactServiceGetOutputArtifact**](ArtifactServiceApi.md#artifactServiceGetOutputArtifact) | **GET** /artifacts/{namespace}/{name}/{nodeId}/{artifactName} | Get an output artifact.
[**artifactServiceGetOutputArtifactByUID**](ArtifactServiceApi.md#artifactServiceGetOutputArtifactByUID) | **GET** /artifacts-by-uid/{uid}/{nodeId}/{artifactName} | Get an output artifact by UID.


<a name="artifactServiceGetInputArtifact"></a>
# **artifactServiceGetInputArtifact**
> artifactServiceGetInputArtifact(namespace, name, nodeId, artifactName)

Get an input artifact.

### Example
```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.auth.*;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:2746");
    
    // Configure API key authorization: BearerToken
    ApiKeyAuth BearerToken = (ApiKeyAuth) defaultClient.getAuthentication("BearerToken");
    BearerToken.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //BearerToken.setApiKeyPrefix("Token");

    ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
    String namespace = "namespace_example"; // String | 
    String name = "name_example"; // String | 
    String nodeId = "nodeId_example"; // String | 
    String artifactName = "artifactName_example"; // String | 
    try {
      apiInstance.artifactServiceGetInputArtifact(namespace, name, nodeId, artifactName);
    } catch (ApiException e) {
      System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetInputArtifact");
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
 **nodeId** | **String**|  |
 **artifactName** | **String**|  |

### Return type

null (empty response body)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

<a name="artifactServiceGetInputArtifactByUID"></a>
# **artifactServiceGetInputArtifactByUID**
> File artifactServiceGetInputArtifactByUID(namespace, uid, nodeId, artifactName)

Get an input artifact by UID.

### Example
```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.auth.*;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:2746");
    
    // Configure API key authorization: BearerToken
    ApiKeyAuth BearerToken = (ApiKeyAuth) defaultClient.getAuthentication("BearerToken");
    BearerToken.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //BearerToken.setApiKeyPrefix("Token");

    ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
    String namespace = "namespace_example"; // String | 
    String uid = "uid_example"; // String | 
    String nodeId = "nodeId_example"; // String | 
    String artifactName = "artifactName_example"; // String | 
    try {
      File result = apiInstance.artifactServiceGetInputArtifactByUID(namespace, uid, nodeId, artifactName);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetInputArtifactByUID");
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
 **uid** | **String**|  |
 **nodeId** | **String**|  |
 **artifactName** | **String**|  |

### Return type

[**File**](File.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

<a name="artifactServiceGetOutputArtifact"></a>
# **artifactServiceGetOutputArtifact**
> File artifactServiceGetOutputArtifact(namespace, name, nodeId, artifactName)

Get an output artifact.

### Example
```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.auth.*;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:2746");
    
    // Configure API key authorization: BearerToken
    ApiKeyAuth BearerToken = (ApiKeyAuth) defaultClient.getAuthentication("BearerToken");
    BearerToken.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //BearerToken.setApiKeyPrefix("Token");

    ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
    String namespace = "namespace_example"; // String | 
    String name = "name_example"; // String | 
    String nodeId = "nodeId_example"; // String | 
    String artifactName = "artifactName_example"; // String | 
    try {
      File result = apiInstance.artifactServiceGetOutputArtifact(namespace, name, nodeId, artifactName);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetOutputArtifact");
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
 **nodeId** | **String**|  |
 **artifactName** | **String**|  |

### Return type

[**File**](File.md)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

<a name="artifactServiceGetOutputArtifactByUID"></a>
# **artifactServiceGetOutputArtifactByUID**
> artifactServiceGetOutputArtifactByUID(uid, nodeId, artifactName)

Get an output artifact by UID.

### Example
```java
// Import classes:
import io.argoproj.workflow.ApiClient;
import io.argoproj.workflow.ApiException;
import io.argoproj.workflow.Configuration;
import io.argoproj.workflow.auth.*;
import io.argoproj.workflow.models.*;
import io.argoproj.workflow.apis.ArtifactServiceApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:2746");
    
    // Configure API key authorization: BearerToken
    ApiKeyAuth BearerToken = (ApiKeyAuth) defaultClient.getAuthentication("BearerToken");
    BearerToken.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //BearerToken.setApiKeyPrefix("Token");

    ArtifactServiceApi apiInstance = new ArtifactServiceApi(defaultClient);
    String uid = "uid_example"; // String | 
    String nodeId = "nodeId_example"; // String | 
    String artifactName = "artifactName_example"; // String | 
    try {
      apiInstance.artifactServiceGetOutputArtifactByUID(uid, nodeId, artifactName);
    } catch (ApiException e) {
      System.err.println("Exception when calling ArtifactServiceApi#artifactServiceGetOutputArtifactByUID");
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
 **uid** | **String**|  |
 **nodeId** | **String**|  |
 **artifactName** | **String**|  |

### Return type

null (empty response body)

### Authorization

[BearerToken](../README.md#BearerToken)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | An artifact file. |  -  |
**0** | An unexpected error response. |  -  |

