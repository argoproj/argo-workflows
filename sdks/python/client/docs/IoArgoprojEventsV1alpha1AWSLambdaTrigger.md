# IoArgoprojEventsV1alpha1AWSLambdaTrigger


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**access_key** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 
**function_name** | **str** | FunctionName refers to the name of the function to invoke. | [optional] 
**invocation_type** | **str** | Choose from the following options.     * RequestResponse (default) - Invoke the function synchronously. Keep    the connection open until the function returns a response or times out.    The API response includes the function response and additional data.     * Event - Invoke the function asynchronously. Send events that fail multiple    times to the function&#39;s dead-letter queue (if it&#39;s configured). The API    response only includes a status code.     * DryRun - Validate parameter values and verify that the user or role    has permission to invoke the function. +optional | [optional] 
**parameters** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) |  | [optional] 
**payload** | [**List[IoArgoprojEventsV1alpha1TriggerParameter]**](IoArgoprojEventsV1alpha1TriggerParameter.md) | Payload is the list of key-value extracted from an event payload to construct the request payload. | [optional] 
**region** | **str** |  | [optional] 
**role_arn** | **str** |  | [optional] 
**secret_key** | [**SecretKeySelector**](SecretKeySelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_argoproj_events_v1alpha1_aws_lambda_trigger import IoArgoprojEventsV1alpha1AWSLambdaTrigger

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojEventsV1alpha1AWSLambdaTrigger from a JSON string
io_argoproj_events_v1alpha1_aws_lambda_trigger_instance = IoArgoprojEventsV1alpha1AWSLambdaTrigger.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojEventsV1alpha1AWSLambdaTrigger.to_json())

# convert the object into a dict
io_argoproj_events_v1alpha1_aws_lambda_trigger_dict = io_argoproj_events_v1alpha1_aws_lambda_trigger_instance.to_dict()
# create an instance of IoArgoprojEventsV1alpha1AWSLambdaTrigger from a dict
io_argoproj_events_v1alpha1_aws_lambda_trigger_form_dict = io_argoproj_events_v1alpha1_aws_lambda_trigger.from_dict(io_argoproj_events_v1alpha1_aws_lambda_trigger_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


