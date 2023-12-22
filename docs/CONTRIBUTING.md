# Contributing

## How To Provide Feedback

Please [raise an issue in Github](https://github.com/argoproj/argo-workflows/issues).

## Code of Conduct

See [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## Community Meetings (monthly)

A monthly opportunity for users and maintainers of Workflows and Events to share their current work and
hear about what’s coming on the roadmap. Please join us! For Community Meeting information, minutes and recordings
please [see here](http://bit.ly/argo-wf-cmty-mtng).

## Contributor Meetings (twice monthly)

A weekly opportunity for committers and maintainers of Workflows and Events to discuss their current work and
talk about what’s next. Feel free to join us! For Contributor Meeting information, minutes and recordings
please [see here](https://bit.ly/argo-data-weekly).

## Slack

You can join the following channels on [CNCF Slack](https://argoproj.github.io/community/join-slack):
* `#argo-workflows`: discussions focused mainly on use of Argo Workflows
* `#argo-wf-contributors`: discussions focused mainly on development of Argo Workflows


## How To Contribute

We're always looking for contributors.

### Authoring PRs

* Documentation - something missing or unclear? Please submit a pull request!
* Code contribution - investigate
  a [good first issue](https://github.com/argoproj/argo-workflows/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22)
  , or anything not assigned.
* You can work on an issue without being assigned.
* Most valuable issues are the ones of higher priority. Priority is indicated by a label of `P0`-`P3`, with lower numbers indicating higher priority.

### Other ways to contribute to the project

* Reviewing PRs
* Responding to questions in the [Slack](#slack) channels
* Responding to questions in [Github Discussions](https://github.com/argoproj/argo-workflows/discussions)
* Triaging new bugs

#### Triaging Bugs

New bugs need to be triaged to identify the highest priority ones.

Apply the labels `P0`, `P1`, `P2`, and `P3`, where `P0` is highest priority and needs immediate attention, followed by `P1`, `P2`, and then `P3`.
If there's a new `P0` bug, notify the [#argo-wf-contributors](https://cloud-native.slack.com/archives/C0510EUH90V) slack channel.

Any bugs with >= 5 "👍" reactions should be labeled at least `P1`.
Any bugs with 3-4 "👍" reactions should be labeled at least `P2`. 
Bugs can be [sorted by "👍"](https://github.com/argoproj/argo-workflows/issues?q=is%3Aissue+is%3Aopen+sort%3Areactions-%2B1-desc+label%3Abug).

If the issue is determined to be a user error and not a bug, remove the `bug` label (and the `regression` label, if applicable) and replace it with the `support` label.
If more information is needed from the author to diagnose the issue, then apply the `more-information-needed` label.


### Roles

The Argo project currently has 4 designated [roles](https://github.com/argoproj/argoproj/blob/main/community/membership.md):
- Member
- Reviewer
- Approver
- Lead

The Reviewer and Approver roles can optionally be scoped to an area of the codebase (for example, UI or docs).


### Sustainability Effort

Argo Workflows is seeking more [Reviewers and Approvers](https://github.com/argoproj/argoproj/blob/main/community/membership.md) to help keep it viable.
Please see [Sustainability Effort](sustainability_effort.md) for more information.

### Running Locally

To run Argo Workflows locally for development: [running locally](running-locally.md).

### Committing

See the [Committing Guidelines](running-locally.md#committing).

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
| It has an acceptable license (e.g. MIT) | ✅ Pass. MIT license.               |
| It is actively maintained.              | ❌ Fail. Project is inactive.        |
| It has no security issues.              | ✅ Pass. No known security issues.  |

No, we should not add that dependency.

### Test Policy

Changes without either unit or e2e tests are unlikely to be accepted.
See [the pull request template](https://github.com/argoproj/argo-workflows/blob/main/.github/pull_request_template.md).

### Contributor Workshop

Please check out the following resources if you are interested in contributing:

* [90m hands-on contributor workshop](https://youtu.be/zZv0lNCDG9w).
* [Deep-dive into components and hands-on experiments](https://docs.google.com/presentation/d/1IU0a3unnr3tBRi38Zn3EHQZj3z6yvocfG9x9icRu1LE/edit?usp=sharing).
* [Architecture overview](https://github.com/argoproj/argo-workflows/blob/main/docs/architecture.md).
