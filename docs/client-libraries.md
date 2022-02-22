# Client Libraries

This page contains an overview of the client libraries for using the Argo API from various programming languages.

To write applications using the REST API, you do not need to implement the API calls and request/response types
yourself. You can use a client library for the programming language you are using.

Client libraries often handle common tasks such as authentication for you.

## Officially-supported client libraries

The following client libraries are officially maintained by the Argo team.

| Language | Client Library | Examples/Docs |
|----------|----------------|---------------|
| Golang   | [apiclient.go](https://github.com/argoproj/argo-workflows/blob/master/pkg/apiclient/apiclient.go) | [Example](https://github.com/argoproj/argo-workflows/blob/master/cmd/argo/commands/submit.go)
| Java     | [Java](https://github.com/argoproj/argo-workflows/blob/master/sdks/java) | |
| Python   | [Python](https://github.com/argoproj/argo-workflows/blob/master/sdks/python) | [Examples](https://github.com/argoproj/argo-workflows/tree/master/sdks/python/examples)/[Docs](https://github.com/argoproj/argo-workflows/tree/master/sdks/python/client/docs) | 

## Community-maintained client libraries

The following client libraries are provided and maintained by their authors, not the Argo team.

| Language | Client Library | Examples/Docs |
|----------|----------------|---------------|
| Python | [Couler](https://github.com/couler-proj/couler) | Multi-workflow engine support Python SDK |
| Python | [Hera](https://github.com/argoproj-labs/hera-workflows) | Easy and accessible Argo workflows construction and submission in Python |

