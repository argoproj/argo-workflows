# IoArgoprojWorkflowV1alpha1CronWorkflowSpec

CronWorkflowSpec is the specification of a CronWorkflow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**concurrency_policy** | **str** | ConcurrencyPolicy is the K8s-style concurrency policy that will be used | [optional] 
**failed_jobs_history_limit** | **int** | FailedJobsHistoryLimit is the number of failed jobs to be kept at a time | [optional] 
**schedule** | **str** | Schedule is a schedule to run the Workflow in Cron format | 
**schedules** | **List[str]** | Schedules is a list of schedules to run the Workflow in Cron format | [optional] 
**starting_deadline_seconds** | **int** | StartingDeadlineSeconds is the K8s-style deadline that will limit the time a CronWorkflow will be run after its original scheduled time if it is missed. | [optional] 
**stop_strategy** | [**IoArgoprojWorkflowV1alpha1StopStrategy**](IoArgoprojWorkflowV1alpha1StopStrategy.md) |  | [optional] 
**successful_jobs_history_limit** | **int** | SuccessfulJobsHistoryLimit is the number of successful jobs to be kept at a time | [optional] 
**suspend** | **bool** | Suspend is a flag that will stop new CronWorkflows from running if set to true | [optional] 
**timezone** | **str** | Timezone is the timezone against which the cron schedule will be calculated, e.g. \&quot;Asia/Tokyo\&quot;. Default is machine&#39;s local time. | [optional] 
**workflow_metadata** | [**ObjectMeta**](ObjectMeta.md) |  | [optional] 
**workflow_spec** | [**IoArgoprojWorkflowV1alpha1WorkflowSpec**](IoArgoprojWorkflowV1alpha1WorkflowSpec.md) |  | 

## Example

```python
from argo_workflows.models.io_argoproj_workflow_v1alpha1_cron_workflow_spec import IoArgoprojWorkflowV1alpha1CronWorkflowSpec

# TODO update the JSON string below
json = "{}"
# create an instance of IoArgoprojWorkflowV1alpha1CronWorkflowSpec from a JSON string
io_argoproj_workflow_v1alpha1_cron_workflow_spec_instance = IoArgoprojWorkflowV1alpha1CronWorkflowSpec.from_json(json)
# print the JSON string representation of the object
print(IoArgoprojWorkflowV1alpha1CronWorkflowSpec.to_json())

# convert the object into a dict
io_argoproj_workflow_v1alpha1_cron_workflow_spec_dict = io_argoproj_workflow_v1alpha1_cron_workflow_spec_instance.to_dict()
# create an instance of IoArgoprojWorkflowV1alpha1CronWorkflowSpec from a dict
io_argoproj_workflow_v1alpha1_cron_workflow_spec_form_dict = io_argoproj_workflow_v1alpha1_cron_workflow_spec.from_dict(io_argoproj_workflow_v1alpha1_cron_workflow_spec_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


