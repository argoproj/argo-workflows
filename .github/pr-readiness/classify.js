'use strict';
// Core decision logic for the PR-readiness helper. Pure functions here are
// unit-tested (test/classify.test.js); the API-calling orchestration lives in
// main.js.

const FAILURE_CONCLUSIONS = new Set(['failure', 'timed_out', 'action_required']);
const NOT_APPLICABLE_CONCLUSIONS = new Set(['skipped', 'cancelled']);

function matchesIgnore(name, patterns) {
  return patterns.some((p) => (p.endsWith('*') ? name.startsWith(p.slice(0, -1)) : name === p));
}

function findRun(checkRuns, match) {
  return checkRuns.find(
    (r) => r.name === match.check && (!match.app || (r.app && r.app.slug === match.app))
  );
}

function runState(run) {
  if (!run) {
    return 'not-applicable';
  }
  if (run.status !== 'completed') {
    return 'pending';
  }
  if (FAILURE_CONCLUSIONS.has(run.conclusion)) {
    return 'failure';
  }
  if (NOT_APPLICABLE_CONCLUSIONS.has(run.conclusion)) {
    return 'not-applicable';
  }
  return 'success'; // success, neutral
}

// Maps the covered signals from checks.config.json onto the live check runs
// for a head SHA. Uncovered checks (unit/E2E tests etc.) are invisible.
function classifySignals(checkRuns, config) {
  return config.signals.map((signal) => {
    const run = findRun(checkRuns, signal.match);
    return {
      id: signal.id,
      title: signal.title,
      guidance: signal.guidance,
      stepGuidance: signal.stepGuidance || null,
      state: runState(run),
      url: run ? run.html_url : null,
    };
  });
}

// Drift detection: failing check runs from apps we cover that match neither a
// signal nor the ignore list — typically a renamed job. Logged, never posted.
classifySignals.diagnostics = function diagnostics(checkRuns, config) {
  const unmapped = checkRuns
    .filter(
      (r) =>
        r.status === 'completed' &&
        FAILURE_CONCLUSIONS.has(r.conclusion) &&
        r.app &&
        config.coveredApps.includes(r.app.slug) &&
        !config.signals.some((s) => findRun([r], s.match)) &&
        !matchesIgnore(r.name, config.ignoreChecks)
    )
    .map((r) => r.name);
  return { unmapped };
};

// The convergence rules. See README.md for the decision table.
function decide({ signals, aiVerdict, existingState, hasExistingComment, pr }) {
  const failing = signals.filter((s) => s.state === 'failure').map((s) => s.id);
  const aiBlocking = Boolean(aiVerdict && aiVerdict.compliant === false);
  const blocking = failing.length > 0 || aiBlocking;
  const anyPending = signals.some((s) => s.state === 'pending');

  let variant = null;
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

  return { variant, shouldComment, shouldDraft, failing, aiBlocking };
}

// OWNERS is a small YAML subset: three keys, each a list of logins.
function parseOwners(yamlText) {
  const sections = new Set(['owners', 'approvers', 'reviewers']);
  const logins = [];
  let current = null;
  for (const line of yamlText.split('\n')) {
    const key = line.match(/^(\w+):/);
    if (key) {
      current = key[1];
      continue;
    }
    const item = line.match(/^-\s*(\S+)/);
    if (item && sections.has(current)) {
      logins.push(item[1]);
    }
  }
  return logins;
}

function isExemptAuthor(user, ownersYaml) {
  if (user.type === 'Bot' || /\[bot\]$/i.test(user.login)) {
    return true;
  }
  const login = user.login.toLowerCase();
  return parseOwners(ownersYaml).some((l) => l.toLowerCase() === login);
}

function findPullRequest(openPrs, headSha) {
  return openPrs.find((pr) => pr.head.sha === headSha) || null;
}

// For checks with per-step guidance (the feature-pr-handling job), pick the
// guidance of the failing step; fall back to the signal's generic guidance.
function pickStepGuidance(signal, steps) {
  if (signal.stepGuidance && Array.isArray(steps)) {
    const failed = steps.find((s) => s.conclusion === 'failure' && signal.stepGuidance[s.name]);
    if (failed) {
      return signal.stepGuidance[failed.name];
    }
  }
  return signal.guidance;
}

module.exports = { classifySignals, decide, parseOwners, isExemptAuthor, findPullRequest, pickStepGuidance };
