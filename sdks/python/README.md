# Argo Python SDK

This is the Python SDK for Argo Workflows.

## Requirements

Python >= 3.6

## Installation

To install the latest development version of the SDK, run the following:

```
pip install git+https://github.com/argoproj/argo-workflows@master#subdirectory=sdks/python/client
```

## Contributing

If you want to release a new version of the SDK you have to first create the git tag:

```shell
git tag [VERSION]
git push origin [VERSION]
```

As an example, 6.0.0 was tagged using `git tag 6.0.0`.

Then, you have to generate the SDK using:

```shell
make generate
```

This will generate the SDK files.

If you have to regenerate the SDK for any reason, you will have to untag the release from a previous commit:
```shell
git tag -d [VERSION]
git tag [VERISON]  # same one, different commit though
```

Then you can regenerate the SDK.

## Getting Started

You can submit a workflow from a raw YAML like the following:

```python
from pprint import pprint

import requests
import yaml

import argo_workflows
from argo_workflows.api import workflow_service_api
from argo_workflows.model.io_argoproj_workflow_v1alpha1_workflow_create_request import
    IoArgoprojWorkflowV1alpha1WorkflowCreateRequest

configuration = argo_workflows.Configuration(host="https://127.0.0.1:2746")
configuration.verify_ssl = False

resp = requests.get('https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/hello-world.yaml')
manifest = yaml.safe_load(resp.text)
manifest['spec']['serviceAccountName'] = 'argo'

api_client = argo_workflows.ApiClient(configuration)
api_instance = workflow_service_api.WorkflowServiceApi(api_client)
api_response = api_instance.create_workflow(
    namespace='argo',
    body=IoArgoprojWorkflowV1alpha1WorkflowCreateRequest(
        workflow=manifest, _check_type=False))
pprint(api_response)
```

Note that `_check_type=False` is required here to avoid type checks against `manifest` which is a Python dictionary.

Alternative, you can submit a workflow with an instance of `IoArgoprojWorkflowV1alpha1Workflow` constructed via the SDK
like the following:

```python
from pprint import pprint

import openapi_client
from openapi_client.api import workflow_service_api
from openapi_client.model.container import Container
from openapi_client.model.io_argoproj_workflow_v1alpha1_template import
    IoArgoprojWorkflowV1alpha1Template
from openapi_client.model.io_argoproj_workflow_v1alpha1_workflow import (
    IoArgoprojWorkflowV1alpha1Template, IoArgoprojWorkflowV1alpha1Workflow
)
from openapi_client.model.io_argoproj_workflow_v1alpha1_workflow_create_request import
    IoArgoprojWorkflowV1alpha1WorkflowCreateRequest
from openapi_client.model.io_argoproj_workflow_v1alpha1_workflow_spec import
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
