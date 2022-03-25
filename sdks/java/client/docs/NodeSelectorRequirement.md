

# NodeSelectorRequirement

A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**key** | **String** | The label key that the selector applies to. | 
**operator** | [**OperatorEnum**](#OperatorEnum) | Represents a key&#39;s relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.  Possible enum values:  - &#x60;\&quot;DoesNotExist\&quot;&#x60;  - &#x60;\&quot;Exists\&quot;&#x60;  - &#x60;\&quot;Gt\&quot;&#x60;  - &#x60;\&quot;In\&quot;&#x60;  - &#x60;\&quot;Lt\&quot;&#x60;  - &#x60;\&quot;NotIn\&quot;&#x60; | 
**values** | **List&lt;String&gt;** | An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch. |  [optional]



## Enum: OperatorEnum

Name | Value
---- | -----
DOESNOTEXIST | &quot;DoesNotExist&quot;
EXISTS | &quot;Exists&quot;
GT | &quot;Gt&quot;
IN | &quot;In&quot;
LT | &quot;Lt&quot;
NOTIN | &quot;NotIn&quot;



