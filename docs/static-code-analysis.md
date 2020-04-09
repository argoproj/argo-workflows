# Static Code Analysis

We use the following static code analysis tools:

* golangci-lint and tslint for compile time linting
* [Github security alerts](https://github.com/argoproj/argo/network/alerts) - for security alerts on dependencies
* [snyk.io](https://app.snyk.io/org/argoproj/projects) - for image scanning
* [sonarcloud.io](https://sonarcloud.io/organizations/argoproj-1/projects) - for security alerts

These are at least run daily or on each pull request.