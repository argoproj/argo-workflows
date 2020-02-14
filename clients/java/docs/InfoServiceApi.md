# InfoServiceApi

All URIs are relative to *http://localhost:2746*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getInfo**](InfoServiceApi.md#getInfo) | **GET** /api/v1/info | 


<a name="getInfo"></a>
# **getInfo**
> InfoInfoResponse getInfo()



### Example
```java
// Import classes:
import io.argoproj.argo.client.ApiClient;
import io.argoproj.argo.client.ApiException;
import io.argoproj.argo.client.Configuration;
import io.argoproj.argo.client.models.*;
import io.argoproj.argo.client.api.InfoServiceApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("http://localhost:2746");

    InfoServiceApi apiInstance = new InfoServiceApi(defaultClient);
    try {
      InfoInfoResponse result = apiInstance.getInfo();
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling InfoServiceApi#getInfo");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**InfoInfoResponse**](InfoInfoResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A successful response. |  -  |

