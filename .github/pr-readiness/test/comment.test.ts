import { test } from 'node:test';
import assert from 'node:assert/strict';
import { MARKER, NOT_READY_LABEL, renderComment } from '../comment.ts';

test('renderComment issues variant lists each failure with guidance and log link', () => {
  const body = renderComment({
    variant: 'issues',
    failures: [
      { id: 'lint', title: 'Lint', guidance: 'Run `make pre-commit -B`.', url: 'https://github.com/x/y/runs/1' },
      { id: 'dco', title: 'DCO (sign-off)', guidance: 'Sign off your commits.', url: 'https://github.com/x/y/runs/2' },
    ],
    templateIssues: null,
    labeled: false,
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
    labeled: false,
  });
  assert.ok(body.includes('<details>'));
  assert.ok(body.includes('Motivation'));
  assert.ok(body.includes('still contains the template placeholder'));
  assert.ok(/waive/i.test(body));
});

test('renderComment notes the not-ready label when labeled', () => {
  const body = renderComment({
    variant: 'issues',
    failures: [{ id: 'lint', title: 'Lint', guidance: 'g', url: 'u' }],
    templateIssues: null,
    labeled: true,
  });
  assert.ok(body.includes(NOT_READY_LABEL));
  assert.ok(/removed automatically/i.test(body));
});

test('renderComment all-clear variant is short and positive', () => {
  const body = renderComment({ variant: 'allclear', failures: [], templateIssues: null, labeled: false });
  assert.ok(body.startsWith(MARKER));
  assert.ok(body.includes('✅'));
  assert.ok(!body.includes('<details>'));
});

test('renderComment waiting variant mentions waiting for checks', () => {
  const body = renderComment({ variant: 'waiting', failures: [], templateIssues: null, labeled: false });
  assert.ok(body.startsWith(MARKER));
  assert.ok(/waiting/i.test(body));
});

test('renderComment footer says tests are not covered and it is automated', () => {
  const body = renderComment({ variant: 'issues', failures: [{ id: 'x', title: 'X', guidance: 'g', url: 'u' }], templateIssues: null, labeled: false });
  assert.ok(/unit\/e2e/i.test(body));
  assert.ok(/automated/i.test(body));
});
