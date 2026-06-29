import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { classifySignals, diagnostics, decide, isExemptAuthor, parseOwners, findPullRequest, pickStepGuidance } from '../classify.ts';
import type { CheckRun, Config, SignalState } from '../types.ts';

const config = createRequire(import.meta.url)('../checks.config.json') as Config;

function run(name: string, conclusion: string | null, status = 'completed', app = 'github-actions'): CheckRun {
  return { name, status, conclusion, html_url: `https://example.invalid/${encodeURIComponent(name)}`, app: { slug: app } };
}

// --- classifySignals ---

test('classifySignals maps failing covered checks to failure with guidance', () => {
  const signals = classifySignals([run('Lint', 'failure'), run('Codegen', 'success')], config);
  const lint = signals.find((s) => s.id === 'lint')!;
  assert.equal(lint.state, 'failure');
  assert.ok(lint.guidance.includes('make pre-commit'));
  assert.ok(lint.url!.includes('Lint'));
  assert.equal(signals.find((s) => s.id === 'codegen')!.state, 'success');
});

test('classifySignals ignores unit/e2e and unknown checks entirely', () => {
  const signals = classifySignals(
    [run('Unit Tests', 'failure'), run('Windows Unit Tests', 'failure'), run('E2E Tests (test-api, mysql, true)', 'failure'), run('E2E Tests - Composite result', 'failure'), run('something-new', 'failure')],
    config
  );
  assert.equal(signals.filter((s) => s.state === 'failure').length, 0);
});

test('classifySignals treats absent, skipped and cancelled checks as not-applicable', () => {
  const signals = classifySignals([run('UI', 'skipped'), run('Codegen', 'cancelled')], config);
  assert.equal(signals.find((s) => s.id === 'ui')!.state, 'not-applicable');
  assert.equal(signals.find((s) => s.id === 'codegen')!.state, 'not-applicable');
  assert.equal(signals.find((s) => s.id === 'lint')!.state, 'not-applicable'); // absent
});

test('classifySignals treats in-progress covered checks as pending', () => {
  const signals = classifySignals([run('Lint', null, 'in_progress')], config);
  assert.equal(signals.find((s) => s.id === 'lint')!.state, 'pending');
});

test('classifySignals covers DCO check from the dco app', () => {
  const signals = classifySignals([run('DCO', 'failure', 'completed', 'dco')], config);
  const dco = signals.find((s) => s.id === 'dco')!;
  assert.equal(dco.state, 'failure');
  assert.ok(/sign.?off/i.test(dco.guidance));
});

test('classifySignals treats timed_out and action_required as failure', () => {
  const signals = classifySignals([run('Lint', 'timed_out'), run('Codegen', 'action_required')], config);
  assert.equal(signals.find((s) => s.id === 'lint')!.state, 'failure');
  assert.equal(signals.find((s) => s.id === 'codegen')!.state, 'failure');
});

test('diagnostics reports unmapped failing checks from covered apps for drift detection', () => {
  const { unmapped } = diagnostics([run('Lint Renamed', 'failure'), run('Unit Tests', 'failure')], config);
  assert.deepEqual(unmapped, ['Lint Renamed']);
});

// --- decide ---

const S = (id: string, state: SignalState) => ({ id, state, title: id, guidance: 'g', url: 'u' });

test('decide: failures present -> issues comment, draft requested', () => {
  const d = decide({
    signals: [S('lint', 'failure'), S('docs', 'pending')],
    templateVerdict: null,
    existingState: null,
    hasExistingComment: false,
    pr: { draft: false, headSha: 'sha1' },
  });
  assert.equal(d.variant, 'issues');
  assert.equal(d.shouldComment, true);
  assert.equal(d.shouldDraft, true);
  assert.deepEqual(d.failing, ['lint']);
});

test('decide: a non-compliant template alone is blocking -> issues + draft', () => {
  const d = decide({
    signals: [S('lint', 'success')],
    templateVerdict: { compliant: false },
    existingState: null,
    hasExistingComment: false,
    pr: { draft: false, headSha: 'sha1' },
  });
  assert.equal(d.variant, 'issues');
  assert.equal(d.shouldDraft, true);
});

test('decide: never post when no failures and no existing comment', () => {
  for (const state of ['pending', 'success'] as SignalState[]) {
    const d = decide({
      signals: [S('lint', state)],
      templateVerdict: { compliant: true },
      existingState: null,
      hasExistingComment: false,
      pr: { draft: false, headSha: 'sha1' },
    });
    assert.equal(d.shouldComment, false, `state=${state}`);
    assert.equal(d.shouldDraft, false);
  }
});

test('decide: existing comment + no failures + pending -> waiting variant', () => {
  const d = decide({
    signals: [S('lint', 'success'), S('docs', 'pending')],
    templateVerdict: null,
    existingState: { draftedSha: null },
    hasExistingComment: true,
    pr: { draft: false, headSha: 'sha1' },
  });
  assert.equal(d.variant, 'waiting');
  assert.equal(d.shouldComment, true);
  assert.equal(d.shouldDraft, false);
});

test('decide: existing comment + all terminal green -> all-clear', () => {
  const d = decide({
    signals: [S('lint', 'success'), S('ui', 'not-applicable')],
    templateVerdict: { compliant: true },
    existingState: { draftedSha: null },
    hasExistingComment: true,
    pr: { draft: false, headSha: 'sha1' },
  });
  assert.equal(d.variant, 'allclear');
  assert.equal(d.shouldComment, true);
  assert.equal(d.shouldDraft, false);
});

test('decide: does not draft a PR that is already a draft', () => {
  const d = decide({
    signals: [S('lint', 'failure')],
    templateVerdict: null,
    existingState: null,
    hasExistingComment: false,
    pr: { draft: true, headSha: 'sha1' },
  });
  assert.equal(d.shouldDraft, false);
  assert.equal(d.shouldComment, true);
});

test('decide: drafts at most once per head SHA (human undraft is respected)', () => {
  const d = decide({
    signals: [S('lint', 'failure')],
    templateVerdict: null,
    existingState: { draftedSha: 'sha1' },
    hasExistingComment: true,
    pr: { draft: false, headSha: 'sha1' },
  });
  assert.equal(d.shouldDraft, false);
  // but a new head re-asserts
  const d2 = decide({
    signals: [S('lint', 'failure')],
    templateVerdict: null,
    existingState: { draftedSha: 'sha1' },
    hasExistingComment: true,
    pr: { draft: false, headSha: 'sha2' },
  });
  assert.equal(d2.shouldDraft, true);
});

test('decide: no failures and a compliant template never drafts', () => {
  const d = decide({
    signals: [S('lint', 'success')],
    templateVerdict: { compliant: true },
    existingState: { draftedSha: null },
    hasExistingComment: true,
    pr: { draft: false, headSha: 'sha1' },
  });
  assert.equal(d.shouldDraft, false);
});

// --- author gating ---

const ownersYaml = ['owners:', '- joibel', '', 'approvers:', '- alexec', '', 'reviewers:', '- blkperl', ''].join('\n');

test('parseOwners extracts all three lists', () => {
  assert.deepEqual(parseOwners(ownersYaml), ['joibel', 'alexec', 'blkperl']);
});

test('isExemptAuthor: OWNERS members, bots and Bot-type users are exempt (case-insensitive)', () => {
  assert.equal(isExemptAuthor({ login: 'Joibel', type: 'User' }, ownersYaml), true);
  assert.equal(isExemptAuthor({ login: 'blkperl', type: 'User' }, ownersYaml), true);
  assert.equal(isExemptAuthor({ login: 'dependabot[bot]', type: 'Bot' }, ownersYaml), true);
  assert.equal(isExemptAuthor({ login: 'renovate[bot]', type: 'User' }, ownersYaml), true);
  // the repo's cherry-pick automation must never burn model quota
  assert.equal(isExemptAuthor({ login: 'argo-cd-cherry-pick-bot[bot]', type: 'Bot' }, ownersYaml), true);
  assert.equal(isExemptAuthor({ login: 'github-actions[bot]', type: 'Bot' }, ownersYaml), true);
  assert.equal(isExemptAuthor({ login: 'random-contributor', type: 'User' }, ownersYaml), false);
});

// --- PR resolution ---

test('findPullRequest matches open PR by head sha', () => {
  const prs = [
    { number: 1, head: { sha: 'aaa' } },
    { number: 2, head: { sha: 'bbb' } },
  ];
  assert.equal(findPullRequest(prs, 'bbb')!.number, 2);
  assert.equal(findPullRequest(prs, 'zzz'), null);
});

// --- pickStepGuidance: choose per-step guidance for the features check ---

const featureSignal = {
  guidance: 'generic feature guidance',
  stepGuidance: {
    'No ./.features/*.md addition': 'run make feature-new',
    'Validate ./.features/*.md changes': 'run make features-validate',
  },
};

test('pickStepGuidance returns the failing step guidance', () => {
  const steps = [
    { name: 'Check if feature PR', conclusion: 'success' },
    { name: 'No ./.features/*.md addition', conclusion: 'failure' },
  ];
  assert.equal(pickStepGuidance(featureSignal, steps), 'run make feature-new');
});

test('pickStepGuidance falls back to generic guidance when no mapped step failed', () => {
  assert.equal(pickStepGuidance(featureSignal, [{ name: 'something else', conclusion: 'failure' }]), 'generic feature guidance');
  assert.equal(pickStepGuidance(featureSignal, null), 'generic feature guidance');
  assert.equal(pickStepGuidance({ guidance: 'g', stepGuidance: null }, []), 'g');
});
