# Client Libraries

This page contains an overview of the client libraries for using the Argo API from various programming languages.

To write applications using the REST API, you do not need to implement the API calls and request/response types
yourself. You can use a client library for the programming language you are using.

Client libraries often handle common tasks such as authentication for you.

## Auto-generated client libraries

The following client libraries are auto-generated using [OpenAPI Generator](https://github.com/OpenAPITools/openapi-generator-cli).
Please expect very minimal support from the Argo team.

| Language | Client Library                                                                                    | Examples/Docs                                                                                                         |
|----------|---------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------|
| Golang   | [`apiclient.go`](https://github.com/argoproj/argo-workflows/blob/main/pkg/apiclient/apiclient.go) | [Example](https://github.com/argoproj/argo-workflows/blob/main/cmd/argo/commands/submit.go)                           |
| Java     | [Java](https://github.com/argoproj/argo-workflows/blob/main/sdks/java)                            |                                                                                                                       |
| Python   | ⚠️ deprecated [Python](https://github.com/argoproj/argo-workflows/blob/main/sdks/python)           | Use [Hera](#hera-python-sdk) instead. Will be removed in version 3.7 |

## Hera Python SDK

Hera is the go-to Python SDK to make Argo Workflows simple and intuitive. It goes beyond a basic REST interface,
allowing you to easily turn Python functions into script templates, and write whole Workflows in Python:

```py
from hera.workflows import DAG, Workflow, script


@script()
def echo(message: str):
    print(message)


with Workflow(
    generate_name="dag-diamond-",
    entrypoint="diamond",
) as w:
    with DAG(name="diamond"):
        A = echo(name="A", arguments={"message": "A"})
        B = echo(name="B", arguments={"message": "B"})
        C = echo(name="C", arguments={"message": "C"})
        D = echo(name="D", arguments={"message": "D"})
        A >> [B, C] >> D  # Define execution order

w.create()
```

Learn more in the [Hera walk-through](https://hera.readthedocs.io/en/stable/walk-through/quick-start/).
