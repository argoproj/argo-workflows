from pprint import pprint

import requests
import yaml

import openapi_client
from openapi_client.api import workflow_service_api
from openapi_client.model.io_argoproj_workflow_v1alpha1_workflow_create_request import \
    IoArgoprojWorkflowV1alpha1WorkflowCreateRequest

configuration = openapi_client.Configuration(host="https://127.0.0.1:2746")
configuration.verify_ssl = False

resp = requests.get('https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/hello-world.yaml')
manifest = yaml.safe_load(resp.text)
manifest['spec']['serviceAccountName'] = 'argo'

api_client = openapi_client.ApiClient(configuration)
api_instance = workflow_service_api.WorkflowServiceApi(api_client)
api_response = api_instance.workflow_service_create_workflow(
	namespace='argo',
	body=IoArgoprojWorkflowV1alpha1WorkflowCreateRequest(
		workflow=manifest, _check_type=False))
pprint(api_response)
