# Argo Workflows Sustainability Effort

Argo Workflows is seeking more community involvement and ultimately more [Reviewers and Approvers](https://github.com/argoproj/argoproj/blob/main/community/membership.md) to help keep it viable. 

We are trying an Incentive System in which contributors pledge a certain number of hours per week (average) and in return have their PRs prioritized for review.

Help is also needed for triaging new incoming bugs by prioritizing them with `P0`, `P1`, `P2`, and `P3` labels.

## Commitment

The Argo project currently has 4 designated [roles](https://github.com/argoproj/argoproj/blob/main/community/membership.md):
- Member
- Reviewer
- Approver
- Lead

For those who focus on just one area of the code (such as UI), the `Reviewer` and `Approver` roles can optionally be scoped to just that area.

Any person in a designated role can sign up to participate in this Incentive System.
Participants are expected to try to advance in roles.
There are different expectations depending on the role:
- Member: should focus primarily on authoring PRs, should average a minimum of 8 hours per week; can also assist with triaging bugs or optionally reviewing PRs
- Reviewer: should author PRs and review PRs to move into the "Approver" role, should average a minimum of 2 hours per week of either PR review time or triaging bugs
- Approver/Lead: should average a minimum of 2 hours per week of either PR review time or triaging bugs

Note that the hours per week from above is an average over time, and it's fine to have weeks of no activity.

Current roles for individuals can be found in [OWNERS](../OWNERS).

Participants should join the [#argo-wf-contributors](https://cloud-native.slack.com/archives/C0510EUH90V) slack channel.

### Finding code to work on

If you have a business need, definitely feel free to work on that.
Otherwise, you can find an issue and assign it to yourself.
If you're a new developer, one option is to pick one up that has the label `good-first-issue`.
You can also try to pick up higher priority issues labeled `P1` or `P2`. 

### Finding a PR to review

First priority is to review any PRs which have the `prioritized` label, meaning they were authored by a participant of this system (verified by GitHub ID).
Each of these should have an Assignee: a Reviewer, Approver, or Lead who "owns" reviewing the PR.<br />
These PRs should be given an initial review within a week.
The Assignee should then respond to each question or modification from the author within a week.

Next, look for PRs with no Assignee. 

If a Reviewer is the Assignee, then once they have approved the PR, they should request a review from one or more Approvers.

### Authoring PRs

Participants can apply a `prioritized` label to any PRs they author.

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
| Isitha Subasinghe         | [isubasinghe](https://github.com/isubasinghe)           |
| Jason Meridth             | [jmeridth](https://github.com/jmeridth)                 |
| Shuangkun Tian            | [shuangkun](https://github.com/shuangkun)               |