# PR Readiness Helper

A standalone bot ([`pr-readiness.yaml`](../workflows/pr-readiness.yaml)) that lowers maintainer burden: when CI finishes on a PR, it keeps **one sticky comment** telling the contributor exactly how to fix contributor-fixable problems, flags not-ready PRs with the **`problem/bot-not-ready`** label, and gets out of the way (clearing the label) once everything is green. The same comment carries a [contributor activity table](#contributor-activity) giving maintainers context on the author.

## What it covers

| Signal | Check run | Guidance |
|---|---|---|
| Lint | `Lint` (CI) | `make pre-commit -B` |
| Codegen | `Codegen` (CI) | `make codegen -B` |
| UI | `UI build and unit tests` (CI) | `yarn --cwd ui …` |
| Build | `Build Binaries (cli/controller)` (CI) | `make cli` / `make controller` |
| Docs | `docs` (Docs) | `make docs` |
| PR title | `title-check` (PR Title Check) | conventional commits |
| Feature file | `feature-pr-handling` (PR Feature Check) | `make feature-new` / `make features-validate` (per failing step) |
| DCO | `DCO` (dco app) | `git rebase --signoff` |
| PR description | template check (no CI signal) | fill in the template |

**Deliberately not covered:** unit tests, Windows unit tests, E2E tests — they are too flaky to be a readiness signal and are never mentioned.

Tune guidance, add or remove signals in [`checks.config.json`](checks.config.json) — it is pure data; no logic changes needed. The `match.check` value is the **check-run name** (= the job's `name:`, or the job *id* when it has none — not a step name).

## Behavior

- Fires on `workflow_run: completed` of CI / Docs / PR Title Check / PR Feature Check, and on `pull_request_target` `opened` / `reopened` / `ready_for_review`. Title and feature checks re-run on PR `edited`, so title/description edits re-evaluate too. `/retest` re-runs CI and therefore re-evaluates.
- Every in-scope PR gets the comment with the [contributor activity table](#contributor-activity), posted when the PR is opened. The readiness section only appears once a covered issue has been found; until then the comment is just the table.
- While issues exist: the readiness section lists only the failing items, each with a fix command and a log link. Pending checks are not mentioned.
- Blocking issues (any covered check failure, or a description that doesn't follow the template) also apply the **`problem/bot-not-ready`** label. The bot owns the label outright: it is applied while the verdict is blocking and **removed automatically** the moment it isn't (fix pushed, check re-run green, description fixed, or the description check waived by a maintainer editing it into compliance). No app token or extra secrets: labelling a PR works with the default Actions token under the `pull-requests: write` permission the sticky comment already needs. (An earlier version converted PRs to draft instead — that needed a dedicated GitHub App because the default token cannot toggle draft state, and undrafting semantics around human intent were messy. The label sidesteps both.)
- Because the label mirrors the current verdict with no memory, a manually-removed label is re-applied on the next CI completion while checks still fail — maintainers who disagree with a covered failure can simply review anyway; the label is advisory, not a gate.
- When issues are resolved but other covered checks are still running: the readiness section shows a short "waiting" state.
- When everything is terminal and green: the readiness section becomes a short ✅ all-clear. A hidden `<!-- pr-readiness:engaged -->` marker in the comment is how later runs know a readiness section was started and must be followed through.
- Skipped: PRs by anyone in [`OWNERS`](../../OWNERS) (owners/approvers/reviewers) and by bots.
- Skipped: PRs that do not target the default branch (`main`), e.g. backports to `release-*`. A stacked PR counts as targeting `main` when its base branch is the head of another open PR that (transitively) targets `main`.

## Contributor activity

Below the readiness section, a table of plain counts for the PR author in this repository — no thresholds or verdicts:

| Contributor activity in argoproj/argo-workflows | |
|---|---:|
| PRs (merged / open / closed) | 2 / 1 / 3 |
| Issues (opened / commented on) | 1 / 4 |
| Others' PRs commented on | 5 |
| Account age | 411 days (since 2025-09-23) |
| Association | First-time contributor |

- One GraphQL query per run ([`contributor.ts`](contributor.ts)): `user.createdAt` plus six aliased `search` counts. "Closed" is closed **without** merging; "open" includes the PR itself.
- "Commented on" counts distinct threads started by someone else (search's `commenter:X -author:X`), not individual comments. Counting every comment would mean paging through the user's whole GitHub comment history, which is unbounded and fails on comments in private repositories.
- "Association" is the PR's `author_association`: *first-time contributor* means first contribution to this repository, *first-time GitHub contributor* first on GitHub at all.
- Recomputed on every run, so the counts stay current; edits to the comment don't notify anyone.
- Best-effort: if the query fails, a warning is logged and the table is left out. If readiness has nothing new to say either, the existing comment is left untouched.

## PR description check

A deterministic check ([`template.ts`](template.ts)) compares the description against [the PR template](../pull_request_template.md): it flags an empty body, the unreplaced `Fixes #TODO` placeholder, and any required `###` section that is missing or left empty (template placeholder comments don't count as content). The required sections are derived from the template itself, so the check follows the template if it changes. It deliberately does **not** judge prose quality.

> An earlier version used GitHub Models (an LLM) to judge description quality. That was removed: GitHub Models is not available to `argoproj` under the CNCF enterprise, and the deterministic check covers the common "didn't fill it in" cases without a model, a network call, data leaving the repo, or `models:` permissions. If Models is ever enabled and richer judgment is wanted, it could be reintroduced behind a flag.

## Security model

- `workflow_run` workflows execute the **default branch's** definition with the base-repo token — a fork PR cannot alter what runs here. `pull_request_target` likewise runs the base branch's definition, never the fork's.
- The job **never checks out or executes PR-head code**; the checkout step pins `ref` to the default branch (without it, `pull_request_target` would check out the PR's base branch). Keep it that way.
- The author's login is checked against GitHub's login format before it is put into a search string.
- `permissions: {}` at the top; the job grants only `pull-requests: write`, `contents: read`, `actions: read`. No secrets beyond `GITHUB_TOKEN`.
- `workflow_run.pull_requests` is empty for fork PRs, so the PR is found by matching `head_sha` against open PRs; no match → exit (a newer push superseded the run).
- PR title/body/branch are attacker-controlled: they are only ever handled as data, never interpolated into shell or scripts. The comment never echoes contributor-supplied text — only the bot's own guidance, check titles/URLs, and template section names.
- All actions are pinned to full commit SHAs (enforced by repo lint).

## Dry run

Setting `DRY_RUN: "true"` in the workflow renders the would-be comment and decisions to the job's **step summary** instead of commenting or labelling. Use it to test changes to the bot against real PRs (correct PR resolution, author gating, sensible text) without posting.

## Maintenance notes

- **Check renamed?** The signal silently stops matching (fail-safe — no false positives) and any failing unmapped check from a covered app is logged as a warning ("unmapped failing check") so you notice. Update `checks.config.json`.
- **Workflow renamed?** Keep `on.workflow_run.workflows` in `pr-readiness.yaml` in sync with the `name:` fields of `ci-build.yaml`, `docs.yaml`, `pr.yaml`, `pr-feature.yaml`.
- **Label renamed?** `NOT_READY_LABEL` in `comment.ts` must match the repo's `problem/bot-not-ready` label exactly — if the label is renamed in repo settings without updating the constant, the add-labels API quietly *recreates* the old name (with a default colour) rather than failing.
- **Known limitation:** first-time contributors whose workflows need approval get only the contributor table and the description check (from the `opened` run) until a maintainer approves the CI run.

## Code & local development

The logic is TypeScript (ESM) under this directory. `main.ts` is the entry point — a single `run()` called from one `actions/github-script` step; the pure, unit-tested logic lives in `classify.ts`, `comment.ts`, `contributor.ts`, `template.ts`, with shared types in `types.ts`.

At runtime there are **no dependencies**: `actions/github-script` runs on Node 24, which strips the TypeScript types when it `require()`s `main.ts` — no build step and nothing compiled is committed. Because stripping does not *type-check*, [`pr-readiness-test.yaml`](../workflows/pr-readiness-test.yaml) runs `tsc --noEmit` and the unit tests on every PR that touches this directory. `typescript` and `@types/node` are dev-only (CI/editor), never shipped to the runner.

```sh
cd .github/pr-readiness
npm ci
npm run typecheck   # tsc --noEmit
npm test            # node --test test/*.test.ts
```
