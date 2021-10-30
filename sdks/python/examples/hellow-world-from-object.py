from pprint import pprint

import openapi_client
from openapi_client.api import workflow_service_api
from openapi_client.model.container import Container
from openapi_client.model.io_argoproj_workflow_v1alpha1_template import \
    IoArgoprojWorkflowV1alpha1Template
from openapi_client.model.io_argoproj_workflow_v1alpha1_workflow import (
    IoArgoprojWorkflowV1alpha1Template, IoArgoprojWorkflowV1alpha1Workflow)
from openapi_client.model.io_argoproj_workflow_v1alpha1_workflow_create_request import \
    IoArgoprojWorkflowV1alpha1WorkflowCreateRequest
from openapi_client.model.io_argoproj_workflow_v1alpha1_workflow_spec import \
    IoArgoprojWorkflowV1alpha1WorkflowSpec
from openapi_client.model.object_meta import ObjectMeta

configuration = openapi_client.Configuration(host="https://127.0.0.1:2746")
configuration.verify_ssl = False

manifest = IoArgoprojWorkflowV1alpha1Workflow(
	metadata=ObjectMeta(generate_name='hello-world-'),
	spec=IoArgoprojWorkflowV1alpha1WorkflowSpec(
		service_account_name='argo',
		entrypoint='whalesay',
		templates=[
		IoArgoprojWorkflowV1alpha1Template(
			name='whalesay',
			container=Container(
				image='docker/whalesay:latest', command=['cowsay'], args=['hello world']))]))

api_client = openapi_client.ApiClient(configuration)
api_instance = workflow_service_api.WorkflowServiceApi(api_client)
api_response = api_instance.workflow_service_create_workflow(
	namespace='argo',
	body=IoArgoprojWorkflowV1alpha1WorkflowCreateRequest(workflow=manifest))
pprint(api_response)
