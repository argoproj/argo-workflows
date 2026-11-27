import { test } from 'node:test';
import assert from 'node:assert/strict';
import { CONTRIBUTOR_MARKER, ENGAGED_MARKER, MARKER, NOT_READY_LABEL, isEngaged, renderComment } from '../comment.ts';

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

const TABLE = '| Contributor activity in x/y | |\n|---|---:|\n| PRs (merged / open / closed) | 0 / 1 / 0 |';

test('renderComment with no variant renders only the contributor table', () => {
  const body = renderComment({ variant: null, failures: [], templateIssues: null, labeled: false, contributorTable: TABLE });
  assert.ok(body.startsWith(MARKER));
  assert.ok(body.includes(CONTRIBUTOR_MARKER));
  assert.ok(body.includes(TABLE));
  assert.ok(!body.includes(ENGAGED_MARKER));
  assert.ok(!/PR readiness/.test(body));
  assert.equal(isEngaged(body), false);
});

test('renderComment puts the readiness section above the contributor table and marks itself engaged', () => {
  const body = renderComment({
    variant: 'issues',
    failures: [{ id: 'lint', title: 'Lint', guidance: 'g', url: 'u' }],
    templateIssues: null,
    labeled: false,
    contributorTable: TABLE,
  });
  assert.ok(body.indexOf('**Lint**') < body.indexOf(TABLE));
  assert.ok(body.indexOf(TABLE) < body.indexOf('Automated PR-readiness helper'));
  assert.equal(isEngaged(body), true);
});

test('renderComment leaves the table out when there is none', () => {
  const body = renderComment({ variant: 'allclear', failures: [], templateIssues: null, labeled: false, contributorTable: null });
  assert.ok(!body.includes(CONTRIBUTOR_MARKER));
  assert.equal(isEngaged(body), true);
});

test('isEngaged treats a comment from before contributor tables as engaged', () => {
  // Such comments only ever existed because something had failed.
  assert.equal(isEngaged(`${MARKER}\n\n#### ⏳ PR readiness\n`), true);
});
