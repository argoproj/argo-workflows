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

An opportunity for contributors and maintainers of Workflows and Events to discuss their current work and talk about what’s next. Feel free to join us!
See the [Contributor Meeting doc](https://bit.ly/argo-data-weekly) for minutes, recordings, and more information.

## Slack

You can join the following channels on [CNCF Slack](https://argoproj.github.io/community/join-slack):

* [`#argo-workflows`](https://cloud-native.slack.com/archives/C01QW9QSSSK): discussions focused mainly on use of Argo Workflows
* [`#argo-wf-contributors`](https://cloud-native.slack.com/archives/C0510EUH90V): discussions focused mainly on development of Argo Workflows

## Roles

The Argo project currently has 4 designated [roles](https://github.com/argoproj/argoproj/blob/main/community/membership.md):

* Member
* Reviewer
* Approver
* Lead

The Reviewer and Approver roles can optionally be scoped to an area of the code base (for example, UI or docs).

Current roles for Reviewers and above can be found in [OWNERS](https://github.com/argoproj/argo-workflows/blob/main/OWNERS).

If you are interested in formally joining the Argo project, [create a Membership request](https://github.com/argoproj/argoproj/issues/new?template=membership.md&title=REQUEST%3A%20New%20membership%20for%20%3Cyour-GH-handle%3E) in the [argoproj](https://github.com/argoproj/argoproj) repository as described in the [Membership](https://github.com/argoproj/argoproj/blob/main/community/membership.md) guide.

## How To Contribute

We're always looking for contributors.

### Authoring PRs

* Documentation - something missing or unclear? Please submit a pull request according to our [docs contribution guide](doc-changes.md)!
* Code contribution - investigate a [good first issue](https://github.com/argoproj/argo-workflows/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22), [high priority bugs](#triaging-bugs), or anything not assigned.
* You can work on an issue without being assigned.

#### Contributor Workshop

Please check out the following resources if you are interested in contributing:

* [90m hands-on contributor workshop](https://youtu.be/zZv0lNCDG9w).
* [Deep-dive into components and hands-on experiments](https://docs.google.com/presentation/d/1IU0a3unnr3tBRi38Zn3EHQZj3z6yvocfG9x9icRu1LE/edit?usp=sharing).
* [Architecture overview](https://github.com/argoproj/argo-workflows/blob/main/docs/architecture.md).

#### Running Locally

To run Argo Workflows locally for development: [running locally](running-locally.md).

#### Committing

See the [Committing Guidelines](running-locally.md#committing).

#### Dependencies

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

#### Test Policy

Changes without either unit or e2e tests are unlikely to be accepted.
See [the pull request template](https://github.com/argoproj/argo-workflows/blob/main/.github/pull_request_template.md).

### Other Contributions

* [Reviewing PRs](#reviewing-prs)
* Responding to questions in the [Slack](#slack) channels
* Responding to questions in [Github Discussions](https://github.com/argoproj/argo-workflows/discussions)
* [Triaging new bugs](#triaging-bugs)

#### Reviewing PRs

Anybody can review a PR.
If you are in a [designated role](#roles), add yourself as an "Assignee" to a PR if you plan to lead the review.
If you are a Reviewer or below, then once you have approved a PR, request a review from one or more Approvers and above.

#### Triaging Bugs

New bugs need to be triaged to identify the highest priority ones.
Any Member can triage bugs.

Apply the labels `P0`, `P1`, `P2`, and `P3`, where `P0` is highest priority and needs immediate attention, followed by `P1`, `P2`, and then `P3`.
If there's a new `P0` bug, notify the [`#argo-wf-contributors`](https://cloud-native.slack.com/archives/C0510EUH90V) Slack channel.

Any bugs with >= 5 "👍" reactions should be labeled at least `P1`.
Any bugs with 3-4 "👍" reactions should be labeled at least `P2`.
Bugs can be [sorted by "👍"](https://github.com/argoproj/argo-workflows/issues?q=is%3Aissue+is%3Aopen+sort%3Areactions-%2B1-desc+label%3Abug).

If the issue is determined to be a user error and not a bug, remove the `bug` label (and the `regression` label, if applicable) and replace it with the `support` label.
If more information is needed from the author to diagnose the issue, then apply the `more information needed` label.

##### Staleness

Only issues and PRs that have the [`more information needed` label](https://github.com/argoproj/argo-workflows/labels/more%20information%20needed) will be considered for staleness.

If the author does not respond timely to a request for more information, the issue or PR will be automatically marked with the `stale` label and a bot message.
Subsequently, if there is still no response, it will be automatically closed as "not planned".

See the [Stale Action configuration](https://github.com/argoproj/argo-workflows/blob/main/.github/workflows/stale.yaml) for more details.

## Sustainability Effort

Argo Workflows is seeking more [Reviewers and Approvers](https://github.com/argoproj/argoproj/blob/main/community/membership.md) to help keep it viable.
Please see [Sustainability Effort](https://github.com/argoproj/argo-workflows/blob/main/community/sustainability_effort.md) for more information.
