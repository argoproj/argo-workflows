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
| Python   | ⚠️ deprecated [Python](https://github.com/argoproj/argo-workflows/blob/main/sdks/python)           | Use one of the [community-maintained](#community-maintained-client-libraries) instead. Will be removed in version 3.7 |

## Community-maintained client libraries

The following client libraries are provided and maintained by their authors, not the Argo team.

| Language | Client Library                                          | Examples/Docs                                                            |
|----------|---------------------------------------------------------|--------------------------------------------------------------------------|
| Python   | [Couler](https://github.com/couler-proj/couler)         | Multi-workflow engine support Python SDK                                 |
| Python   | [Hera](https://github.com/argoproj-labs/hera-workflows) | Easy and accessible Argo workflows construction and submission in Python |
