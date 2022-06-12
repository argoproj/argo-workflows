# Contributing

## How To Provide Feedback

Please [raise an issue in Github](https://github.com/argoproj/argo-workflows/issues).

## Code of Conduct

See [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## Contributor Meetings

A weekly opportunity for committers and maintainers of Workflows, Events, and Dataflow to discuss their current work and
talk about what’s next. Feel free to join us! For Contributor Meeting information, minutes and recordings
please [see here](https://bit.ly/argo-data-weekly).

## How To Contribute

We're always looking for contributors.

* Documentation - something missing or unclear? Please submit a pull request!
* Code contribution - investigate
  a [good first issue](https://github.com/argoproj/argo-workflows/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22)
  , or anything not assigned.
* Join the `#argo-contributors` channel on [our Slack](https://argoproj.github.io/community/join-slack).
* Get a [mentor](mentoring.md) to help you get started.

### Running Locally

To run Argo Workflows locally for development: [running locally](running-locally.md).

### Dependencies

Dependencies increase the risk of security issues and have on-going maintenance costs.

The dependency must pass these test:

* A strong use case.
* It has an acceptable license (e.g. MIT).
* It is actively maintained.
* It has no security issues.

Example, should we add `fasttemplate`
, [view the Snyk report](https://snyk.io/advisor/golang/github.com/valyala/fasttemplate):

| Test                                    | Outcome                             |
|-----------------------------------------|-------------------------------------|
| A strong use case.                      | ❌ Fail. We can use `text/template`. |
| It has an acceptable license (e.g. MIT) | ✅ Pass. MIT license.                |
| It is actively maintained.              | ❌ Fail. Project is inactive.        |
| It has no security issues.              | ✅ Pass. No known security issues.   |

No, we should not add that dependency.

### Test Policy

Changes without either unit or e2e tests are unlikely to be accepted.
See [the pull request template](https://github.com/argoproj/argo-workflows/blob/master/.github/pull_request_template.md)
.

### Contributor Workshop

We have a [90m video on YouTube](https://youtu.be/zZv0lNCDG9w) show you have to get hands-on contributing.
