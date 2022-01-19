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

import argo_workflows
from argo_workflows.api import workflow_service_api
from argo_workflows.model.io_argoproj_workflow_v1alpha1_workflow_create_request import \
    IoArgoprojWorkflowV1alpha1WorkflowCreateRequest

configuration = argo_workflows.Configuration(host="https://127.0.0.1:2746")
configuration.verify_ssl = False

resp = requests.get('https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/hello-world.yaml')
manifest = yaml.safe_load(resp.text)

api_client = argo_workflows.ApiClient(configuration)
api_instance = workflow_service_api.WorkflowServiceApi(api_client)
api_response = api_instance.create_workflow(
    namespace="argo",
    body=IoArgoprojWorkflowV1alpha1WorkflowCreateRequest(workflow=manifest, _check_return_type=False, _check_type=False))
pprint(api_response)

```

Note that `_check_type=False` is required here to avoid type checks against `manifest` which is a Python dictionary and `_check_return_type=False` avoids [an existing issue](https://github.com/argoproj/argo-workflows/issues/7293) with OpenAPI generator.

Alternative, you can submit a workflow with an instance of `IoArgoprojWorkflowV1alpha1Workflow` constructed via the SDK
like the following:

```python
from pprint import pprint

import argo_workflows
from argo_workflows.api import workflow_service_api
from argo_workflows.model.container import Container
from argo_workflows.model.io_argoproj_workflow_v1alpha1_template import IoArgoprojWorkflowV1alpha1Template
from argo_workflows.model.io_argoproj_workflow_v1alpha1_workflow import IoArgoprojWorkflowV1alpha1Workflow
from argo_workflows.model.io_argoproj_workflow_v1alpha1_workflow_create_request import
    IoArgoprojWorkflowV1alpha1WorkflowCreateRequest
from argo_workflows.model.io_argoproj_workflow_v1alpha1_workflow_spec import
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
api_instance = workflow_service_api.WorkflowServiceApi(api_client)

if __name__ == '__main__':
    api_response = api_instance.create_workflow(
        namespace='argo',
        body=IoArgoprojWorkflowV1alpha1WorkflowCreateRequest(workflow=manifest, _check_return_type=False))
    pprint(api_response)

```

## Examples

You can find additional examples [here](examples).

## API Reference

You can find the API reference [here](client/docs).
