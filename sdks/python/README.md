# Argo Python SDK

This is the Python SDK for Argo Workflows.

## Requirements

Python >= 3.6

## Installation

To install the latest development version of the SDK, run the following:

```
pip install git+https://github.com/argoproj/argo-workflows@master#subdirectory=sdks/python/client
```

## Getting Started

You can submit a workflow from a raw YAML like the following:

```python
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
```

Note that `_check_type=False` is required here to avoid type checks against `manifest` which is a Python dictionary.

Alternative, you can submit a workflow with an instance of `IoArgoprojWorkflowV1alpha1Workflow` constructed via the SDK like the following:

```python
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
```

## Examples

You can find additional examples [here](examples).

## API Reference

You can find the API reference [here](client/docs).
