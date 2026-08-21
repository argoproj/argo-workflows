// Orchestration for the PR Readiness Helper workflow. A single entry point,
// run(), called from one actions/github-script step in pr-readiness.yaml:
// resolve the PR, gate the author, classify checks, check the description
// against the template, sync the not-ready label, and render the sticky
// comment (or the dry-run summary). All decision logic lives in the
// unit-tested modules this file imports.

import * as fs from 'node:fs';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

import { classifySignals, diagnostics, decide, isExemptAuthor, findPullRequest, pickStepGuidance } from './classify.ts';
import { MARKER, NOT_READY_LABEL, renderComment } from './comment.ts';
import { checkTemplate } from './template.ts';
import type { Config, JobStep } from './types.ts';

// Minimal structural types for the actions/github-script bindings we use, so
// the bot keeps zero runtime dependencies (no @actions/* packages).
interface Summary {
  addHeading(text: string, level?: number): Summary;
  addRaw(text: string): Summary;
  addTable(rows: unknown[]): Summary;
  write(): Promise<unknown>;
}
interface Core {
  info(message: string): void;
  warning(message: string): void;
  setOutput(name: string, value: string): void;
  startGroup(name: string): void;
  endGroup(): void;
  summary: Summary;
}
interface RepoContext {
  repo: { owner: string; repo: string };
  payload: { workflow_run: { head_sha: string } };
}
interface Octokit {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  paginate(route: unknown, params: Record<string, unknown>): Promise<any[]>;
  rest: {
    actions: { getJobForWorkflowRun(params: Record<string, unknown>): Promise<{ data: { steps?: JobStep[] } }> };
    pulls: { list: unknown };
    checks: { listForRef: unknown };
    issues: {
      listComments: unknown;
      addLabels(params: Record<string, unknown>): Promise<unknown>;
      removeLabel(params: Record<string, unknown>): Promise<unknown>;
    };
  };
}

const here = path.dirname(fileURLToPath(import.meta.url));
const config = JSON.parse(fs.readFileSync(path.join(here, 'checks.config.json'), 'utf8')) as Config;

function tempDir(): string {
  const dir = path.join(process.env.RUNNER_TEMP || '/tmp', 'pr-readiness');
  fs.mkdirSync(dir, { recursive: true });
  return dir;
}

function errMessage(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

export async function run({ github, context, core }: { github: Octokit; context: RepoContext; core: Core }): Promise<void> {
  const { owner, repo } = context.repo;
  const headSha = context.payload.workflow_run.head_sha;
  const dryRun = process.env.DRY_RUN === 'true';
  const stop = (reason: string): void => {
    core.info(`Skipping: ${reason}`);
    core.setOutput('proceed', 'false');
    core.setOutput('should_comment', 'false');
  };

  // Resolve the PR by head SHA — workflow_run.pull_requests is empty for
  // fork PRs, so list open PRs and match instead.
  const openPrs = await github.paginate(github.rest.pulls.list, { owner, repo, state: 'open', per_page: 100 });
  const pr = findPullRequest(openPrs, headSha);
  if (!pr) {
    return stop(`no open PR with head ${headSha} (superseded by a newer push, or closed)`);
  }

  // Maintainers and bots help themselves. OWNERS is read from the default
  // branch (the workflow only ever checks out the default branch).
  const ownersYaml = fs.readFileSync('OWNERS', 'utf8');
  if (isExemptAuthor(pr.user, ownersYaml)) {
    return stop(`author ${pr.user.login} is exempt (OWNERS member or bot)`);
  }

  // Classify all check runs on the head SHA (covers CI, Docs, title, feature
  // and the DCO app; unit/E2E checks are deliberately not covered).
  const checkRuns = await github.paginate(github.rest.checks.listForRef, { owner, repo, ref: headSha, per_page: 100 });
  const signals = classifySignals(checkRuns, config);
  const { unmapped } = diagnostics(checkRuns, config);

  // Step-level guidance for the feature check: the check-run id doubles as
  // the Actions job id, whose steps tell us which validation failed.
  const featureSignal = signals.find((s) => s.id === 'features');
  if (featureSignal && featureSignal.state === 'failure') {
    const featureRun = checkRuns.find((r) => r.name === 'feature-pr-handling');
    if (featureRun) {
      try {
        const { data: job } = await github.rest.actions.getJobForWorkflowRun({ owner, repo, job_id: featureRun.id });
        featureSignal.guidance = pickStepGuidance(featureSignal, job.steps ?? null);
      } catch (e) {
        core.warning(`could not fetch feature job steps: ${errMessage(e)}`); // generic guidance still applies
      }
    }
  }

  // Find our existing sticky comment (if any). Author check matters: anyone
  // can paste our marker into a comment, but only the actions bot's comment
  // may be trusted as ours.
  const comments = await github.paginate(github.rest.issues.listComments, { owner, repo, issue_number: pr.number, per_page: 100 });
  const existing = comments.find(
    (c) => c.user && c.user.login === 'github-actions[bot]' && typeof c.body === 'string' && c.body.includes(MARKER)
  );

  // Deterministic PR-description / template check (no model required).
  const template = fs.readFileSync('.github/pull_request_template.md', 'utf8');
  const templateVerdict = checkTemplate(pr.body || '', template);

  const decision = decide({
    signals,
    templateVerdict,
    hasExistingComment: Boolean(existing),
  });

  // Sync the not-ready label to the verdict: the bot owns the label, so it is
  // applied while blocking and removed once not. Best-effort: a label API
  // failure never blocks the comment.
  const hadLabel = Array.isArray(pr.labels) && pr.labels.some((l: { name?: string }) => l.name === NOT_READY_LABEL);
  let labeled = dryRun ? decision.blocking : hadLabel; // dry run previews the would-be state
  if (!dryRun && decision.blocking !== hadLabel) {
    try {
      if (decision.blocking) {
        await github.rest.issues.addLabels({ owner, repo, issue_number: pr.number, labels: [NOT_READY_LABEL] });
      } else {
        await github.rest.issues.removeLabel({ owner, repo, issue_number: pr.number, name: NOT_READY_LABEL });
      }
      labeled = decision.blocking;
    } catch (e) {
      core.warning(`could not ${decision.blocking ? 'add' : 'remove'} label '${NOT_READY_LABEL}' on PR #${pr.number}: ${errMessage(e)}`);
    }
  }

  const state = { v: 1, failing: decision.failing };

  const commentBody = renderComment({
    variant: decision.variant,
    failures: signals.filter((s) => decision.failing.includes(s.id)),
    templateIssues: decision.templateBlocking ? templateVerdict.issues : null,
    labeled,
    state,
  });

  for (const name of unmapped) {
    core.warning(`unmapped failing check (rename? update checks.config.json): ${name}`);
  }

  core.info(
    `PR #${pr.number} by ${pr.user.login} head=${headSha} | signals: ` +
      signals.map((s) => `${s.id}=${s.state}`).join(' ') +
      ` | template=${templateVerdict.compliant ? 'ok' : 'issues'}` +
      ` | comment=${decision.shouldComment} variant=${decision.variant || 'n/a'} blocking=${decision.blocking} label: ${hadLabel} -> ${labeled}`
  );
  if (decision.shouldComment) {
    core.startGroup('rendered comment');
    core.info(commentBody);
    core.endGroup();
  }

  if (dryRun) {
    core.summary
      .addHeading('PR Readiness Helper — dry run', 3)
      .addRaw(`PR: #${pr.number} · head: \`${headSha}\` · would comment: **${decision.shouldComment}**` +
        ` (variant: ${decision.variant || 'n/a'}) · label \`${NOT_READY_LABEL}\`: ${hadLabel} → would be ${decision.blocking}\n\n`)
      .addRaw(decision.shouldComment ? '#### Rendered comment\n\n' + commentBody + '\n' : '')
      .addTable([
        [{ data: 'signal', header: true }, { data: 'state', header: true }],
        ...signals.map((s) => [s.id, s.state]),
      ]);
    if (unmapped.length > 0) {
      core.summary.addRaw(`\n⚠️ Unmapped failing checks: ${unmapped.join(', ')}\n`);
    }
    await core.summary.write();
    core.setOutput('proceed', 'true');
    core.setOutput('should_comment', 'false');
    core.setOutput('pr_number', String(pr.number));
    return;
  }

  if (decision.shouldComment) {
    fs.writeFileSync(path.join(tempDir(), 'comment.md'), commentBody);
  }
  core.setOutput('proceed', 'true');
  core.setOutput('should_comment', String(decision.shouldComment));
  core.setOutput('pr_number', String(pr.number));
}
