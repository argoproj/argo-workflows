# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TriggerParameterSource


## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**context_key** | **str** | ContextKey is the JSONPath of the event&#39;s (JSON decoded) context key ContextKey is a series of keys separated by a dot. A key may contain wildcard characters &#39;*&#39; and &#39;?&#39;. To access an array value use the index as the key. The dot and wildcard characters can be escaped with &#39;\\\\&#39;. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. | [optional] 
**context_template** | **str** |  | [optional] 
**data_key** | **str** | DataKey is the JSONPath of the event&#39;s (JSON decoded) data key DataKey is a series of keys separated by a dot. A key may contain wildcard characters &#39;*&#39; and &#39;?&#39;. To access an array value use the index as the key. The dot and wildcard characters can be escaped with &#39;\\\\&#39;. See https://github.com/tidwall/gjson#path-syntax for more information on how to use this. | [optional] 
**data_template** | **str** |  | [optional] 
**dependency_name** | **str** | DependencyName refers to the name of the dependency. The event which is stored for this dependency is used as payload for the parameterization. Make sure to refer to one of the dependencies you have defined under Dependencies list. | [optional] 
**use_raw_data** | **bool** |  | [optional] 
**value** | **str** | Value is the default literal value to use for this parameter source This is only used if the DataKey is invalid. If the DataKey is invalid and this is not defined, this param source will produce an error. | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


