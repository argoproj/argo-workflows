# IoK8sApiPolicyV1PodDisruptionBudgetSpec

PodDisruptionBudgetSpec is a description of a PodDisruptionBudget.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**max_unavailable** | **str** |  | [optional] 
**min_available** | **str** |  | [optional] 
**selector** | [**LabelSelector**](LabelSelector.md) |  | [optional] 

## Example

```python
from argo_workflows.models.io_k8s_api_policy_v1_pod_disruption_budget_spec import IoK8sApiPolicyV1PodDisruptionBudgetSpec

# TODO update the JSON string below
json = "{}"
# create an instance of IoK8sApiPolicyV1PodDisruptionBudgetSpec from a JSON string
io_k8s_api_policy_v1_pod_disruption_budget_spec_instance = IoK8sApiPolicyV1PodDisruptionBudgetSpec.from_json(json)
# print the JSON string representation of the object
print(IoK8sApiPolicyV1PodDisruptionBudgetSpec.to_json())

# convert the object into a dict
io_k8s_api_policy_v1_pod_disruption_budget_spec_dict = io_k8s_api_policy_v1_pod_disruption_budget_spec_instance.to_dict()
# create an instance of IoK8sApiPolicyV1PodDisruptionBudgetSpec from a dict
io_k8s_api_policy_v1_pod_disruption_budget_spec_form_dict = io_k8s_api_policy_v1_pod_disruption_budget_spec.from_dict(io_k8s_api_policy_v1_pod_disruption_budget_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


