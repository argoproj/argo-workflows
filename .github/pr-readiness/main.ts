// Orchestration for the PR Readiness Helper workflow. A single entry point,
// run(), called from one actions/github-script step in pr-readiness.yaml:
// resolve the PR, gate the author, classify checks, check the description
// against the template, convert to draft if blocking, and render the sticky
// comment (or the dry-run summary). All decision logic lives in the
// unit-tested modules this file imports.

import * as fs from 'node:fs';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

import { classifySignals, diagnostics, decide, isExemptAuthor, findPullRequest, pickStepGuidance } from './classify.ts';
import { MARKER, renderComment, parseState } from './comment.ts';
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
    issues: { listComments: unknown };
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

  // Find our existing sticky comment (if any) and recover its state blob.
  // Author check matters: anyone can paste our marker into a comment, but
  // only the actions bot's comment may be trusted as state.
  const comments = await github.paginate(github.rest.issues.listComments, { owner, repo, issue_number: pr.number, per_page: 100 });
  const existing = comments.find(
    (c) => c.user && c.user.login === 'github-actions[bot]' && typeof c.body === 'string' && c.body.includes(MARKER)
  );
  const existingState = existing ? parseState(existing.body) : null;

  // Deterministic PR-description / template check (no model required).
  const template = fs.readFileSync('.github/pull_request_template.md', 'utf8');
  const templateVerdict = checkTemplate(pr.body || '', template);

  const decision = decide({
    signals,
    templateVerdict,
    existingState,
    hasExistingComment: Boolean(existing),
    pr: { draft: pr.draft, headSha },
  });

  // Draft conversion: at most once per head SHA; undrafting is human-only.
  // The default Actions token cannot toggle draft state ("Resource not
  // accessible by integration"), so this requires the app token minted by
  // the workflow. Best-effort: failure never blocks the comment.
  let draftedNow = false;
  if (decision.shouldDraft && !dryRun) {
    const token = process.env.DRAFT_TOKEN;
    if (!token) {
      core.warning(
        `PR #${pr.number} should be drafted, but no draft token is available ` +
          '(PR_READINESS_APP_ID / PR_READINESS_APP_PRIVATE_KEY secrets not configured?)'
      );
    } else {
      try {
        const res = await fetch('https://api.github.com/graphql', {
          method: 'POST',
          headers: { authorization: `bearer ${token}`, 'content-type': 'application/json' },
          body: JSON.stringify({
            query: 'mutation($id: ID!) { convertPullRequestToDraft(input: {pullRequestId: $id}) { pullRequest { isDraft } } }',
            variables: { id: pr.node_id },
          }),
        });
        const result = (await res.json()) as { errors?: Array<{ message: string }> };
        if (!res.ok || result.errors) {
          throw new Error(result.errors ? result.errors.map((e) => e.message).join('; ') : `HTTP ${res.status}`);
        }
        draftedNow = true;
      } catch (e) {
        core.warning(`could not convert PR #${pr.number} to draft: ${errMessage(e)}`);
      }
    }
  }

  const state = {
    v: 1,
    failing: decision.failing,
    draftedSha: draftedNow ? headSha : (existingState && existingState.draftedSha) || null,
  };

  const commentBody = renderComment({
    variant: decision.variant,
    failures: signals.filter((s) => decision.failing.includes(s.id)),
    templateIssues: decision.templateBlocking ? templateVerdict.issues : null,
    drafted: draftedNow,
    state,
  });

  for (const name of unmapped) {
    core.warning(`unmapped failing check (rename? update checks.config.json): ${name}`);
  }

  core.info(
    `PR #${pr.number} by ${pr.user.login} head=${headSha} | signals: ` +
      signals.map((s) => `${s.id}=${s.state}`).join(' ') +
      ` | template=${templateVerdict.compliant ? 'ok' : 'issues'}` +
      ` | comment=${decision.shouldComment} variant=${decision.variant || 'n/a'} draft=${decision.shouldDraft} draftedNow=${draftedNow}`
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
        ` (variant: ${decision.variant || 'n/a'}) · would draft: **${decision.shouldDraft}**\n\n`)
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
