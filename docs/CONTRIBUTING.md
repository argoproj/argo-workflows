# Contributing

## How To Provide Feedback

Please [raise an issue in Github](https://github.com/argoproj/argo/issues).

## Code of Conduct

See [code of conduct](../CODE_OF_CONDUCT.md).

## How To Contribute

We're always looking for contributors. 

* Documentation - something missing or unclear? Please submit a pull request!
* Code contribution - investigate a [help wanted issue](https://github.com/argoproj/argo/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22+label%3A%22good+first+issue%22), or anything labelled with "good first issue"?
* Join the #argo-devs channel on [our Slack](https://argoproj.github.io/community/join-slack).

### Running Locally

To run Argo Workflows locally for development: [running locally](running-locally.md).

### Test Policy

Changes without either unit or e2e tests are unlikely to be accepted. See [the pull request template](../.github/pull_request_template.md.)

### Running Sonar Locally

Install the scanner:

```
brew install sonar-scanner
```

Run the tests:

```
make test CI=true
make test-reports/test-report.out
```

Perform a scan:

```
# the key is PR number (e.g. "2666"), the branch is the CI branch, e.g. "pull/2666"
SONAR_TOKEN=... sonar-scanner -Dsonar.pullrequest.key=... -Dsonar.pullrequest.branch=... 
```
 