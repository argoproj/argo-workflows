from pprint import pprint

import argo_workflows
from argo_workflows.api import workflow_service_api
from argo_workflows.model.container import Container
from argo_workflows.model.io_argoproj_workflow_v1alpha1_template import IoArgoprojWorkflowV1alpha1Template
from argo_workflows.model.io_argoproj_workflow_v1alpha1_workflow import IoArgoprojWorkflowV1alpha1Workflow
from argo_workflows.model.io_argoproj_workflow_v1alpha1_workflow_create_request import \
    IoArgoprojWorkflowV1alpha1WorkflowCreateRequest
from argo_workflows.model.io_argoproj_workflow_v1alpha1_workflow_spec import \
    IoArgoprojWorkflowV1alpha1WorkflowSpec
from argo_workflows.model.object_meta import ObjectMeta

configuration = argo_workflows.Configuration(host="https://127.0.0.1:2746")
configuration.verify_ssl = False

manifest = IoArgoprojWorkflowV1alpha1Workflow(
    metadata=ObjectMeta(generate_name='hello-world-'),
    spec=IoArgoprojWorkflowV1alpha1WorkflowSpec(
        entrypoint='whalesay',
        templates=[
            IoArgoprojWorkflowV1alpha1Template(
                name='whalesay',
                container=Container(
                    image='docker/whalesay:latest', command=['cowsay'], args=['hello world']))]))

api_client = argo_workflows.ApiClient(configuration)
api_instance = workflow_service_api.WorkflowServiceApi(api_client=api_client)
api_response = api_instance.create_workflow(
    namespace='argo',
    body=IoArgoprojWorkflowV1alpha1WorkflowCreateRequest(workflow=manifest),
    _check_return_type=False)
pprint(api_response)
