# Client Libraries

This page contains an overview of the client libraries for using the Argo API from various programming languages.

* [Officially-supported client libraries](#officially-supported-client-libraries)
* [Community-maintained client libraries](#community-maintained-client-libraries)

To write applications using the REST API, you do not need to implement the API calls and request/response types
yourself. You can use a client library for the programming language you are using.

Client libraries often handle common tasks such as authentication for you.

## Officially-supported client libraries

The following client libraries are officially maintained by the Argo team.

| Language | Client Library | Examples/Docs |
|----------|----------------|---------------|
| Golang   | [apiclient.go](https://github.com/argoproj/argo-workflows/blob/master/pkg/apiclient/apiclient.go) | [Example](https://github.com/argoproj/argo-workflows/blob/master/cmd/argo/commands/submit.go)
| Java     | [java](java) | [Examples/Docs](https://github.com/argoproj/argo-workflows/blob/master/pkg/apiclient/apiclient.go) | [Example](https://github.com/argoproj/argo-workflows/blob/master/sdks/java/generated/docs) |
| Python   | [python](python) | TBC | 

## Community-maintained client libraries

The following client libraries are provided and maintained by their authors, not the Argo team.

| Language | Client Library | Examples/Docs |
|----------|----------------|---------------|
| Python | [Couler](https://github.com/couler-proj/couler) | Multi-workflow engine support |
| Python | Hera | TBC |
