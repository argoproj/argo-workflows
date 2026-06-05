'use strict';
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { neutralizeMentions, redactClosingKeywords, sanitizeAiText } = require('../sanitize');

test('neutralizeMentions wraps @mentions in backticks so GitHub does not notify', () => {
  assert.equal(neutralizeMentions('ping @alice please'), 'ping `@alice` please');
  assert.equal(neutralizeMentions('@argoproj/argo-workflows-approvers look'), '`@argoproj/argo-workflows-approvers` look');
});

test('neutralizeMentions leaves emails and already-escaped mentions alone', () => {
  assert.equal(neutralizeMentions('mail alan@clucas.org'), 'mail alan@clucas.org');
  assert.equal(neutralizeMentions('`@alice`'), '`@alice`');
});

test('redactClosingKeywords removes issue-closing references', () => {
  assert.equal(redactClosingKeywords('this Fixes #123 ok'), 'this [issue ref removed] ok');
  assert.equal(redactClosingKeywords('Closes #1 and resolves #22'), '[issue ref removed] and [issue ref removed]');
  assert.equal(redactClosingKeywords('fixed #9'), '[issue ref removed]');
});

test('redactClosingKeywords leaves plain issue numbers and words alone', () => {
  assert.equal(redactClosingKeywords('see #123'), 'see #123');
  assert.equal(redactClosingKeywords('fixes nothing'), 'fixes nothing');
});

test('sanitizeAiText applies both sanitizers and caps length', () => {
  const out = sanitizeAiText('hi @bob, fixes #5 ' + 'x'.repeat(500), 100);
  assert.ok(out.includes('`@bob`'));
  assert.ok(!/fixes #5/i.test(out));
  assert.ok(out.length <= 100 + 1); // +1 for truncation ellipsis
  assert.ok(out.endsWith('…'));
});

test('sanitizeAiText returns short text unchanged apart from sanitizing', () => {
  assert.equal(sanitizeAiText('all good', 100), 'all good');
});
