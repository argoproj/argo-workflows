# Argo Workflows Sustainability Effort

Argo Workflows is seeking more community involvement and ultimately more [Reviewers and Approvers](https://github.com/argoproj/argoproj/blob/main/community/membership.md) to help keep it viable. 

We are trying out a system in which users pledge a certain number of hours per week (average) and in return have their own PRs prioritized for review.

Help is also needed for triaging new incoming bugs (prioritizing them with `P0`, `P1`, `P2`, and `P3` labels).

## Commitment

The Argo project currently has 4 designated [roles](https://github.com/argoproj/argoproj/blob/main/community/membership.md):
- Member
- Reviewer
- Approver
- Lead

For those who focus on just one area of the code (such as UI), the `Reviewer` and `Approver` roles can optionally be scoped to just that area.

Anybody in any of these roles can sign up to be a part of this Incentive System. Participants are expected to try to advance in roles. The minimum expectations are different depending on which role:
- Member: should focus primarily on authoring PRs, should average a minimum of 8 hours per week; can also assist with triaging bugs
- Reviewer: should author PRs and review PRs to move into the "Approver" role, should average a minimum of 2 hours per week of either PR review time or triaging bugs
- Approver/Lead: should average a minimum of 2 hours per week of either PR review time or triaging bugs

Note that the hours per week from above is an average over time, and it's fine to have weeks of no activity.

Current roles for individuals can be found in [this document](https://github.com/argoproj/argoproj/blob/main/MAINTAINERS.md).

Participants should join the [#argo-wf-contributors](https://cloud-native.slack.com/archives/C0510EUH90V) slack channel.

### Finding code to work on

If you have a business need, definitely feel free to work on that. Otherwise, you can find an Issue and assign it to yourself. If you're a new developer, one option is to pick one up that has the 
label `good-first-issue`. Otherwise, you can pick up an Issue labeled `P1` or `P2`. 

### Finding a PR to review

We will first prioritize review of any PRs which have the "prioritized" label, meaning they were authored by a participant of this sytem (we can verify that they in fact are by github ID). We need to make 
sure each of these has an Assignee. An Assignee is a Reviewer/Approver/Lead who owns reviewing the PR. PRs with this label should be given an initial review within a week and should respond to each question 
or modification from the author within a week.

Next, look for the Pull Requests which have no existing Assignee. 

In the case of a Reviewer being the Assignee, once the Reviewer has approved the PR, they can request a review from one or more Approvers.

### Authoring PRs

For participants of the system, any PRs that you author you can apply a "prioritized" label to.

### Bug Triaging

We need to make sure that all new bugs are seen by somebody so that we can identify the highest priority ones. There are labels `P0`, `P1`, `P2`, and `P3` that should be applied, in which `P0` is considered
highest priority and needs immediate attention, followed by `P1`, `P2`, and then `P3`. The "bug" label can be removed if the issue is determined to be user error. The label `more-information-needed` can be added 
if more information is needed from the author to in order to determine whether it's a bug or its priority. If there's a new `P0` bug, notify the [#argo-wf-contributors](https://cloud-native.slack.com/archives/C0510EUH90V) slack channel.

Any bugs that have >= 5 "thumbs up" reactions should be labeled `P1`. Any bugs with 3-4 "thumbs up" should be labeled `P2`. (Bugs can be sorted by "thumbs up").

## Participants

(if you'd like to join, just add your name here and submit in a PR)

| Name                      | GitHub ID                                               |
|---------------------------|---------------------------------------------------------|
| Julie Vogelman            | [juliev0](https://github.com/juliev0)                   |
| Saravanan Balasubramanian | [sarabala1979](https://github.com/sarabala1979)         |
| Yuan Tang                 | [terrytangyuan](https://github.com/terrytangyuan)       |
| Alan Clucas               | [Joibel](https://github.com/Joibel)                     |