# PR Readiness Helper

A standalone bot ([`pr-readiness.yaml`](../workflows/pr-readiness.yaml)) that lowers maintainer burden: when CI finishes on a PR, it keeps **one sticky comment** telling the contributor exactly how to fix contributor-fixable problems, moves not-ready PRs to **draft**, and gets out of the way once everything is green.

## What it covers

| Signal | Check run | Guidance |
|---|---|---|
| Lint | `Lint` (CI) | `make pre-commit -B` |
| Codegen | `Codegen` (CI) | `make codegen -B` |
| UI | `UI` (CI) | `yarn --cwd ui …` |
| Build | `Build Binaries (cli/controller)` (CI) | `make cli` / `make controller` |
| Docs | `docs` (Docs) | `make docs` |
| PR title | `title-check` (PR Title Check) | conventional commits |
| Feature file | `feature-pr-handling` (PR Feature Check) | `make feature-new` / `make features-validate` (per failing step) |
| DCO | `DCO` (dco app) | `git rebase --signoff` |
| PR description | GitHub Models verdict | fill in the template |

**Deliberately not covered:** unit tests, Windows unit tests, E2E tests — they are too flaky to be a readiness signal and are never mentioned.

Tune guidance, add or remove signals in [`checks.config.json`](checks.config.json) — it is pure data; no logic changes needed. The `match.check` value is the **check-run name** (= the job's `name:`, or the job *id* when it has none — not a step name).

## Behavior

- Fires on `workflow_run: completed` of CI / Docs / PR Title Check / PR Feature Check. Title and feature checks re-run on PR `edited`, so title/description edits re-evaluate too. `/retest` re-runs CI and therefore re-evaluates.
- **Never posts** on a PR that never had a covered issue.
- While issues exist: one comment listing only the failing items, each with a fix command and a log link. Pending checks are not mentioned.
- Blocking issues (any covered check failure, or a confident "template not filled in" AI verdict) also **convert the PR to draft**. The bot **never** marks ready-for-review — that's the contributor's call — and it drafts at most once per head SHA, so a human re-marking it ready is respected until new commits arrive.
- When issues are resolved but other covered checks are still running: the comment shows a short "waiting" state.
- When everything is terminal and green: the comment is edited to a short ✅ all-clear.
- Skipped: PRs by anyone in [`OWNERS`](../../OWNERS) (owners/approvers/reviewers) and by bots.

## PR description check (GitHub Models)

Deterministic shortcut first (empty body needs no model), then a body-hash cache (the verdict is reused until the description changes — the hash lives in a hidden state blob in the comment), then `actions/ai-inference` with [`ai-prompt.md`](ai-prompt.md). The verdict must match [`ai-schema.json`](ai-schema.json) exactly (enforced fail-closed in [`ai.js`](ai.js)); anything else — including rate limits and outages — silently drops the AI layer for that run. Model output is sanitized (`@mentions` neutralized, `fixes #N` redacted, length-capped) before it is posted.

## Security model

- `workflow_run` workflows execute the **default branch's** definition with the base-repo token — a fork PR cannot alter what runs here.
- The job **never checks out or executes PR-head code**; the checkout step takes the default branch only. Keep it that way.
- `permissions: {}` at the top; the job grants only `pull-requests: write`, `contents: read`, `actions: read`, `models: read`. No secrets beyond `GITHUB_TOKEN`.
- `workflow_run.pull_requests` is empty for fork PRs, so the PR is found by matching `head_sha` against open PRs; no match → exit (a newer push superseded the run).
- PR title/body/branch are attacker-controlled: they are only ever handled as data (env vars / files), never interpolated into shell or scripts.
- The PR body fed to the model is untrusted; the prompt instructs the model to ignore instructions inside it, and the output is schema-validated and sanitized regardless.
- All actions are pinned to full commit SHAs (enforced by repo lint).

## Dry run & rollout

`DRY_RUN: "true"` in the workflow renders the would-be comment and decisions to the job's **step summary** instead of commenting or drafting. Roll out: merge with dry-run on → watch summaries on real PRs (correct PR resolution, author gating, sensible text) → flip to `"false"`.

## Maintenance notes

- **Check renamed?** The signal silently stops matching (fail-safe — no false positives) and any failing unmapped check from a covered app is logged as a warning ("unmapped failing check") so you notice. Update `checks.config.json`.
- **Workflow renamed?** Keep `on.workflow_run.workflows` in `pr-readiness.yaml` in sync with the `name:` fields of `ci-build.yaml`, `docs.yaml`, `pr.yaml`, `pr-feature.yaml`.
- **Known limitation:** first-time contributors whose workflows need approval get no help until a maintainer approves the run (nothing completes, so nothing fires).
- Tests: `node --test '.github/pr-readiness/test/*.test.js'` (plain `node:test`, no dependencies).
