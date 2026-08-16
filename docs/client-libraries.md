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
* [Typescript](#juno-typescript-sdk)

Please feel free to contribute more language libraries to help improve the Argo Workflows ecosystem.

### Go SDK

The [Go SDK](./go-sdk-guide.md) is a fully-featured client for Argo Workflows. It provides two client approaches:

* **Kubernetes Client** - Direct CRD access for in-cluster applications
* **Argo Server Client** - gRPC/HTTP access for remote applications

**Documentation:**

* [Go SDK Guide](./go-sdk-guide.md) - Comprehensive documentation
* [Examples](https://github.com/argoproj/argo-workflows/blob/main/sdks/go/) - Working code examples
* [API Reference](https://pkg.go.dev/github.com/argoproj/argo-workflows/v4)

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

### Juno Typescript SDK

Juno is a workflow generator that allows you to write your Workflows in Typescript. Juno reduces your usage of pass by string and provides types and validation to make writing complex Workflows just a little less painful.

Juno is a community-supported project

```ts
import { Arguments } from '../src/api/arguments';
import { DagTask } from '../src/api/dag-task';
import { DagTemplate } from '../src/api/dag-template';
import { Inputs } from '../src/api/inputs';
import { InputParameter } from '../src/api/parameter';
import { Template } from '../src/api/template';
import { Workflow } from '../src/api/workflow';
import { WorkflowSpec } from '../src/api/workflow-spec';
import { IoArgoprojWorkflowV1Alpha1Workflow } from '../src/workflow-interfaces/data-contracts';
import { and, simpleTag } from '../src/api/expression';
import { Container } from '../src/api/container';

export async function generateTemplate(): Promise<IoArgoprojWorkflowV1Alpha1Workflow> {
    const messageInputParameter = new InputParameter('message');

    const echoTemplateInputs = new Inputs({
        parameters: [messageInputParameter],
    });

    const echoTemplate = new Template('echo', {
        container: new Container({
            command: ['echo', simpleTag(echoTemplateInputs.parameters?.[0] as InputParameter)],
            image: 'alpine:3.7',
        }),
        inputs: echoTemplateInputs,
    });

    const taskA = new DagTask('A', {
        arguments: new Arguments({
            parameters: [messageInputParameter.toArgumentParameter({ value: 'A' })],
        }),
        template: echoTemplate,
    });

    const taskB = new DagTask('B', {
        arguments: new Arguments({
            parameters: [messageInputParameter.toArgumentParameter({ value: 'B' })],
        }),
        depends: taskA,
        template: echoTemplate,
    });

    const taskC = new DagTask('C', {
        arguments: new Arguments({
            parameters: [messageInputParameter.toArgumentParameter({ value: 'C' })],
        }),
        depends: taskA,
        template: echoTemplate,
    });

    const diamondTemplate = new Template('diamond', {
        dag: new DagTemplate({
            tasks: [
                taskA,
                taskB,
                taskC,
                new DagTask('D', {
                    arguments: new Arguments({
                        parameters: [messageInputParameter.toArgumentParameter({ value: 'D' })],
                    }),
                    depends: and([taskB, taskC]),
                    template: echoTemplate,
                }),
            ],
        }),
    });

    return new Workflow({
        metadata: {
            generateName: 'dag-diamond-',
        },
        spec: new WorkflowSpec({
            entrypoint: diamondTemplate,
        }),
    }).toWorkflow();
}
```

Learn more at [Juno Docs](https://github.com/lumindigital/juno/blob/main/docs/index.md).
