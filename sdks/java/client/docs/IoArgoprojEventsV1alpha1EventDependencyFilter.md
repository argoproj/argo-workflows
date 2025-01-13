

# IoArgoprojEventsV1alpha1EventDependencyFilter

EventDependencyFilter defines filters and constraints for a io.argoproj.workflow.v1alpha1.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**context** | [**IoArgoprojEventsV1alpha1EventContext**](IoArgoprojEventsV1alpha1EventContext.md) |  |  [optional]
**data** | [**List&lt;IoArgoprojEventsV1alpha1DataFilter&gt;**](IoArgoprojEventsV1alpha1DataFilter.md) |  |  [optional]
**dataLogicalOperator** | **String** | DataLogicalOperator defines how multiple Data filters (if defined) are evaluated together. Available values: and (&amp;&amp;), or (||) Is optional and if left blank treated as and (&amp;&amp;). |  [optional]
**exprLogicalOperator** | **String** | ExprLogicalOperator defines how multiple Exprs filters (if defined) are evaluated together. Available values: and (&amp;&amp;), or (||) Is optional and if left blank treated as and (&amp;&amp;). |  [optional]
**exprs** | [**List&lt;IoArgoprojEventsV1alpha1ExprFilter&gt;**](IoArgoprojEventsV1alpha1ExprFilter.md) | Exprs contains the list of expressions evaluated against the event payload. |  [optional]
**script** | **String** | Script refers to a Lua script evaluated to determine the validity of an io.argoproj.workflow.v1alpha1. |  [optional]
**time** | [**IoArgoprojEventsV1alpha1TimeFilter**](IoArgoprojEventsV1alpha1TimeFilter.md) |  |  [optional]



