import { test } from 'node:test';
import assert from 'node:assert/strict';
import { MARKER, renderComment, parseState } from '../comment.ts';
import type { State } from '../types.ts';

const baseState: State = { v: 1, failing: ['lint'], draftedSha: null };

test('renderComment issues variant lists each failure with guidance and log link', () => {
  const body = renderComment({
    variant: 'issues',
    failures: [
      { id: 'lint', title: 'Lint', guidance: 'Run `make pre-commit -B`.', url: 'https://github.com/x/y/runs/1' },
      { id: 'dco', title: 'DCO (sign-off)', guidance: 'Sign off your commits.', url: 'https://github.com/x/y/runs/2' },
    ],
    templateIssues: null,
    holdingDraft: false,
    needsReadyForReview: false,
    state: baseState,
  });
  assert.ok(body.startsWith(MARKER), 'starts with hidden marker');
  assert.ok(body.includes('**Lint**'));
  assert.ok(body.includes('Run `make pre-commit -B`.'));
  assert.ok(body.includes('https://github.com/x/y/runs/1'));
  assert.ok(body.includes('**DCO (sign-off)**'));
  assert.ok(!body.includes('✅ PR readiness: all clear'));
});

test('renderComment includes template findings in a waivable details block', () => {
  const body = renderComment({
    variant: 'issues',
    failures: [],
    templateIssues: [{ section: 'Motivation', problem: 'still contains the template placeholder' }],
    holdingDraft: false,
    needsReadyForReview: false,
    state: baseState,
  });
  assert.ok(body.includes('<details>'));
  assert.ok(body.includes('Motivation'));
  assert.ok(body.includes('still contains the template placeholder'));
  assert.ok(/waive/i.test(body));
});

test('renderComment explains the draft on every render while the bot holds it', () => {
  const args = {
    variant: 'issues' as const,
    failures: [{ id: 'lint', title: 'Lint', guidance: 'g', url: 'u' }],
    templateIssues: null,
    needsReadyForReview: false,
    state: baseState,
  };
  // the note is a function of holding the draft, not of having just imposed
  // it: the comment is edited in place, so a note that renders only on the
  // drafting run is gone by the time anyone reads it
  const held = renderComment({ ...args, holdingDraft: true });
  assert.ok(/draft/i.test(held));
  assert.ok(/Ready for review/.test(held));
  assert.ok(!/\[!NOTE\]/.test(renderComment({ ...args, holdingDraft: false })));
});

test('renderComment carries the draft note into the waiting variant', () => {
  const body = renderComment({
    variant: 'waiting',
    failures: [],
    templateIssues: null,
    holdingDraft: true,
    needsReadyForReview: false,
    state: { ...baseState, failing: [] },
  });
  assert.ok(/waiting/i.test(body));
  assert.ok(/draft/i.test(body));
  assert.ok(/Ready for review/.test(body));
});

test('renderComment all-clear variant is short and positive', () => {
  const body = renderComment({ variant: 'allclear', failures: [], templateIssues: null, holdingDraft: false, needsReadyForReview: false, state: { ...baseState, failing: [] } });
  assert.ok(body.startsWith(MARKER));
  assert.ok(body.includes('✅'));
  assert.ok(!body.includes('<details>'));
});

test('renderComment all-clear asks for Ready for review when the bot could not lift its draft', () => {
  const body = renderComment({
    variant: 'allclear',
    failures: [],
    templateIssues: null,
    holdingDraft: false,
    needsReadyForReview: true,
    state: { ...baseState, failing: [] },
  });
  assert.ok(body.includes('✅'));
  assert.ok(/Ready for review/.test(body));
  // must not tell them to sit back: nobody reviews a draft
  assert.ok(!/take it from here/.test(body));
});

test('renderComment waiting variant mentions waiting for checks', () => {
  const body = renderComment({ variant: 'waiting', failures: [], templateIssues: null, holdingDraft: false, needsReadyForReview: false, state: baseState });
  assert.ok(body.startsWith(MARKER));
  assert.ok(/waiting/i.test(body));
});

test('renderComment footer says tests are not covered and it is automated', () => {
  const body = renderComment({ variant: 'issues', failures: [{ id: 'x', title: 'X', guidance: 'g', url: 'u' }], templateIssues: null, holdingDraft: false, needsReadyForReview: false, state: baseState });
  assert.ok(/unit\/e2e/i.test(body));
  assert.ok(/automated/i.test(body));
});

test('state round-trips through the rendered comment', () => {
  const state: State = { v: 1, failing: ['lint', 'dco'], draftedSha: 'cafe01' };
  const body = renderComment({ variant: 'issues', failures: [{ id: 'lint', title: 'L', guidance: 'g', url: 'u' }], templateIssues: null, holdingDraft: false, needsReadyForReview: false, state });
  assert.deepEqual(parseState(body), state);
});

test('parseState returns null for non-bot or malformed comments', () => {
  assert.equal(parseState('just a human comment'), null);
  assert.equal(parseState(MARKER + '\n<!-- state: {not json} -->'), null);
});
