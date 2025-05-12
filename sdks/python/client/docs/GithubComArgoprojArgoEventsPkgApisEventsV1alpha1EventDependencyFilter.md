# GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventDependencyFilter

EventDependencyFilter defines filters and constraints for a io.argoproj.workflow.v1alpha1.

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**context** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventContext**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1EventContext.md) |  | [optional] 
**data** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1DataFilter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1DataFilter.md) |  | [optional] 
**data_logical_operator** | **str** | DataLogicalOperator defines how multiple Data filters (if defined) are evaluated together. Available values: and (&amp;&amp;), or (||) Is optional and if left blank treated as and (&amp;&amp;). | [optional] 
**expr_logical_operator** | **str** | ExprLogicalOperator defines how multiple Exprs filters (if defined) are evaluated together. Available values: and (&amp;&amp;), or (||) Is optional and if left blank treated as and (&amp;&amp;). | [optional] 
**exprs** | [**[GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ExprFilter]**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1ExprFilter.md) | Exprs contains the list of expressions evaluated against the event payload. | [optional] 
**script** | **str** | Script refers to a Lua script evaluated to determine the validity of an io.argoproj.workflow.v1alpha1. | [optional] 
**time** | [**GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TimeFilter**](GithubComArgoprojArgoEventsPkgApisEventsV1alpha1TimeFilter.md) |  | [optional] 
**any string name** | **bool, date, datetime, dict, float, int, list, str, none_type** | any string name can be used but the value must be the correct type | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


