# Argo Python SDK

This is the Python SDK for Argo Workflows.

## Requirements

Python >= 3.6

## Installation & Usage

To install the latest development version of the SDK, run the following:

```
pip install git+https://github.com/argoproj/argo-workflows@master#subdirectory=sdks/python/client
```

Then import the package:
```python
import openapi_client
```

## Getting Started

Please follow the [installation procedure](#installation--usage) and then run the following:

```python
import time
import openapi_client
import yaml
import requests

from pprint import pprint
from openapi_client.api import workflow_service_api
from openapi_client.model.io_argoproj_workflow_v1alpha1_workflow_create_request import IoArgoprojWorkflowV1alpha1WorkflowCreateRequest

configuration = openapi_client.Configuration(host="http://localhost:2746")

resp = requests.get('https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/dag-diamond-steps.yaml')
manifest = yaml.safe_load(resp.text)
api_client = openapi_client.ApiClient(configuration)
api_instance = workflow_service_api.WorkflowServiceApi(api_client)
api_response = api_instance.workflow_service_create_workflow('argo', IoArgoprojWorkflowV1alpha1WorkflowCreateRequest(workflow=manifest, _check_type=False))
pprint(api_response)
```
