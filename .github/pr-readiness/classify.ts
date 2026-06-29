// Core decision logic for the PR-readiness helper. Pure functions here are
// unit-tested (test/classify.test.ts); the API-calling orchestration lives in
// main.ts.

import type { CheckRun, Config, Decision, GitHubUser, JobStep, Signal, SignalMatch, SignalState } from './types.ts';

const FAILURE_CONCLUSIONS = new Set(['failure', 'timed_out', 'action_required']);
const NOT_APPLICABLE_CONCLUSIONS = new Set(['skipped', 'cancelled']);

function matchesIgnore(name: string, patterns: string[]): boolean {
  return patterns.some((p) => (p.endsWith('*') ? name.startsWith(p.slice(0, -1)) : name === p));
}

function findRun(checkRuns: CheckRun[], match: SignalMatch): CheckRun | undefined {
  return checkRuns.find(
    (r) => r.name === match.check && (!match.app || (r.app != null && r.app.slug === match.app))
  );
}

function runState(run: CheckRun | undefined): SignalState {
  if (!run) {
    return 'not-applicable';
  }
  if (run.status !== 'completed') {
    return 'pending';
  }
  if (run.conclusion !== null && FAILURE_CONCLUSIONS.has(run.conclusion)) {
    return 'failure';
  }
  if (run.conclusion !== null && NOT_APPLICABLE_CONCLUSIONS.has(run.conclusion)) {
    return 'not-applicable';
  }
  return 'success'; // success, neutral
}

// Maps the covered signals from checks.config.json onto the live check runs
// for a head SHA. Uncovered checks (unit/E2E tests etc.) are invisible.
export function classifySignals(checkRuns: CheckRun[], config: Config): Signal[] {
  return config.signals.map((signal) => {
    const run = findRun(checkRuns, signal.match);
    return {
      id: signal.id,
      title: signal.title,
      guidance: signal.guidance,
      stepGuidance: signal.stepGuidance ?? null,
      state: runState(run),
      url: run ? run.html_url : null,
    };
  });
}

// Drift detection: failing check runs from apps we cover that match neither a
// signal nor the ignore list — typically a renamed job. Logged, never posted.
export function diagnostics(checkRuns: CheckRun[], config: Config): { unmapped: string[] } {
  const unmapped = checkRuns
    .filter(
      (r) =>
        r.status === 'completed' &&
        r.conclusion !== null &&
        FAILURE_CONCLUSIONS.has(r.conclusion) &&
        r.app != null &&
        config.coveredApps.includes(r.app.slug) &&
        !config.signals.some((s) => findRun([r], s.match)) &&
        !matchesIgnore(r.name, config.ignoreChecks)
    )
    .map((r) => r.name);
  return { unmapped };
}

interface DecideArgs {
  signals: ReadonlyArray<{ id: string; state: string }>;
  templateVerdict: { compliant: boolean } | null;
  existingState: { draftedSha?: string | null } | null;
  hasExistingComment: boolean;
  pr: { draft: boolean; headSha: string };
}

// The convergence rules. See README.md for the decision table.
export function decide({ signals, templateVerdict, existingState, hasExistingComment, pr }: DecideArgs): Decision {
  const failing = signals.filter((s) => s.state === 'failure').map((s) => s.id);
  const templateBlocking = Boolean(templateVerdict && templateVerdict.compliant === false);
  const blocking = failing.length > 0 || templateBlocking;
  const anyPending = signals.some((s) => s.state === 'pending');

  let variant: Decision['variant'] = null;
  let shouldComment = false;
  if (blocking) {
    variant = 'issues';
    shouldComment = true;
  } else if (hasExistingComment) {
    variant = anyPending ? 'waiting' : 'allclear';
    shouldComment = true;
  }

  const alreadyDraftedThisSha = Boolean(existingState && existingState.draftedSha === pr.headSha);
  const shouldDraft = blocking && !pr.draft && !alreadyDraftedThisSha;

  return { variant, shouldComment, shouldDraft, failing, templateBlocking };
}

// OWNERS is a small YAML subset: three keys, each a list of logins.
export function parseOwners(yamlText: string): string[] {
  const sections = new Set(['owners', 'approvers', 'reviewers']);
  const logins: string[] = [];
  let current: string | null = null;
  for (const line of yamlText.split('\n')) {
    const key = line.match(/^(\w+):/);
    if (key) {
      current = key[1];
      continue;
    }
    const item = line.match(/^-\s*(\S+)/);
    if (item && current !== null && sections.has(current)) {
      logins.push(item[1]);
    }
  }
  return logins;
}

export function isExemptAuthor(user: GitHubUser, ownersYaml: string): boolean {
  if (user.type === 'Bot' || /\[bot\]$/i.test(user.login)) {
    return true;
  }
  const login = user.login.toLowerCase();
  return parseOwners(ownersYaml).some((l) => l.toLowerCase() === login);
}

export function findPullRequest<T extends { head: { sha: string } }>(openPrs: T[], headSha: string): T | null {
  return openPrs.find((pr) => pr.head.sha === headSha) ?? null;
}

// For checks with per-step guidance (the feature-pr-handling job), pick the
// guidance of the failing step; fall back to the signal's generic guidance.
export function pickStepGuidance(
  signal: { guidance: string; stepGuidance?: Record<string, string> | null },
  steps: JobStep[] | null
): string {
  if (signal.stepGuidance && Array.isArray(steps)) {
    const failed = steps.find((s) => s.conclusion === 'failure' && signal.stepGuidance![s.name]);
    if (failed) {
      return signal.stepGuidance[failed.name];
    }
  }
  return signal.guidance;
}
