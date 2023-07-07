# IoArgoprojEventsV1alpha1EventDependencyFilter

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**context** | Option<[**crate::models::IoArgoprojEventsV1alpha1EventContext**](io.argoproj.events.v1alpha1.EventContext.md)> |  | [optional]
**data** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1DataFilter>**](io.argoproj.events.v1alpha1.DataFilter.md)> |  | [optional]
**data_logical_operator** | Option<**String**> | DataLogicalOperator defines how multiple Data filters (if defined) are evaluated together. Available values: and (&&), or (||) Is optional and if left blank treated as and (&&). | [optional]
**expr_logical_operator** | Option<**String**> | ExprLogicalOperator defines how multiple Exprs filters (if defined) are evaluated together. Available values: and (&&), or (||) Is optional and if left blank treated as and (&&). | [optional]
**exprs** | Option<[**Vec<crate::models::IoArgoprojEventsV1alpha1ExprFilter>**](io.argoproj.events.v1alpha1.ExprFilter.md)> | Exprs contains the list of expressions evaluated against the event payload. | [optional]
**script** | Option<**String**> | Script refers to a Lua script evaluated to determine the validity of an io.argoproj.workflow.v1alpha1. | [optional]
**time** | Option<[**crate::models::IoArgoprojEventsV1alpha1TimeFilter**](io.argoproj.events.v1alpha1.TimeFilter.md)> |  | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


