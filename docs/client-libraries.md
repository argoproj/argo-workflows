# Client Libraries

This page contains an overview of the client libraries for using the Argo API from various programming languages.

To write applications using the REST API, you do not need to implement the API calls and request/response types
yourself. You can use a client library for the programming language you are using.

Client libraries often handle common tasks such as authentication for you.

## Client Libraries

We have libraries for the following languages:

* [Go](#go-sdk)
* [Java](#java-sdk)
* [Python](#hera-python-sdk)

Please feel free to contribute more language libraries to help improve the Argo Workflows ecosystem.

### Go SDK

The [Go SDK](./go-sdk-guide.md) is a fully-featured client for Argo Workflows. It provides two client approaches:

* **Kubernetes Client** - Direct CRD access for in-cluster applications
* **Argo Server Client** - gRPC/HTTP access for remote applications

**Documentation:**

* [Go SDK Guide](./go-sdk-guide.md) - Comprehensive documentation
* [Examples](https://github.com/argoproj/argo-workflows/blob/main/sdks/go/) - Working code examples
* [API Reference](https://pkg.go.dev/github.com/argoproj/argo-workflows/v3)

### Java SDK

The [Java](https://github.com/argoproj/argo-workflows/blob/main/sdks/java) library is auto-generated using OpenAPI Generator.
It is community supported.

### Hera Python SDK

Hera is the recommended Python SDK for Argo Workflows. It makes Argo Workflows simple and intuitive, going beyond a basic REST interface to allow you to easily turn Python functions into script templates and write whole Workflows in Python:

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
