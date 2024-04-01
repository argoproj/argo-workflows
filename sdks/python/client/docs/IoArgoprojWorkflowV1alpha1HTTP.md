# IoArgoprojWorkflowV1alpha1HTTP


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**body** | **str** | Body is content of the HTTP Request | [optional] 
**body_from** | [**IoArgoprojWorkflowV1alpha1HTTPBodySource**](IoArgoprojWorkflowV1alpha1HTTPBodySource.md) |  | [optional] 
**headers** | [**List[IoArgoprojWorkflowV1alpha1HTTPHeader]**](IoArgoprojWorkflowV1alpha1HTTPHeader.md) | Headers are an optional list of headers to send with HTTP requests | [optional] 
**insecure_skip_verify** | **bool** | InsecureSkipVerify is a bool when if set to true will skip TLS verification for the HTTP client | [optional] 
**method** | **str** | Method is HTTP methods for HTTP Request | [optional] 
**success_condition** | **str** | SuccessCondition is an expression if evaluated to true is considered successful | [optional] 
**timeout_seconds** | **int** | TimeoutSeconds is request timeout for HTTP Request. Default is 30 seconds | [optional] 
**url** | **str** | URL of the HTTP Request | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_http import IoArgoprojWorkflowV1alpha1HTTP

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1HTTP from a JSON string
io_argoproj_workflow_v1alpha1_http_instance = IoArgoprojWorkflowV1alpha1HTTP.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1HTTP.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_http_dict = io_argoproj_workflow_v1alpha1_http_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1HTTP from a dict
io_argoproj_workflow_v1alpha1_http_form_dict = io_argoproj_workflow_v1alpha1_http.from_dict(io_argoproj_workflow_v1alpha1_http_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


