import { test } from 'node:test';
import assert from 'node:assert/strict';
import { checkTemplate } from '../template.ts';

// A template with the same shape as .github/pull_request_template.md: a
// multi-line comment block (containing its own headers, which must NOT be
// treated as required sections), a Fixes placeholder, and five sections.
const TEMPLATE = [
  '<!--',
  '### Before you open your PR',
  '- run make pre-commit',
  '### When you open your PR',
  '- mark as draft',
  '-->',
  '',
  '<!-- Does this PR fix an issue -->',
  'Fixes #TODO',
  '',
  '### Motivation',
  '<!-- TODO: Say why you made your changes. -->',
  '',
  '### Modifications',
  '<!-- TODO: Say what changes you made. -->',
  '',
  '### Verification',
  '<!-- TODO: Say how you tested your changes. -->',
  '',
  '### Documentation',
  '<!-- TODO: docs. -->',
  '',
  '### AI',
  '<!-- TODO: Declare any use of AI. Say "None" if no AI was used. -->',
  '',
].join('\n');

const FILLED = [
  'Fixes #5',
  '',
  '### Motivation',
  'Because the thing was broken.',
  '',
  '### Modifications',
  'Fixed the thing.',
  '',
  '### Verification',
  'Ran the e2e tests.',
  '',
  '### Documentation',
  'Not needed — internal change.',
  '',
  '### AI',
  'None.',
  '',
].join('\n');

function sectionsOf(issues: { section: string }[]): string[] {
  return issues.map((i) => i.section);
}

test('a fully filled-in description is compliant', () => {
  const v = checkTemplate(FILLED, TEMPLATE);
  assert.equal(v.compliant, true);
  assert.deepEqual(v.issues, []);
});

test('an empty description is not compliant', () => {
  const v = checkTemplate('', TEMPLATE);
  assert.equal(v.compliant, false);
  assert.equal(v.issues.length, 1);
});

test('the unedited template (all placeholders) is not compliant', () => {
  const v = checkTemplate(TEMPLATE, TEMPLATE);
  assert.equal(v.compliant, false);
  // Fixes placeholder + every empty section
  assert.ok(sectionsOf(v.issues).includes('Fixes'));
  assert.ok(sectionsOf(v.issues).includes('Motivation'));
  assert.ok(sectionsOf(v.issues).includes('AI'));
});

test('headers inside the comment block are not treated as required sections', () => {
  const v = checkTemplate(FILLED, TEMPLATE);
  assert.ok(!sectionsOf(v.issues).includes('Before you open your PR'));
  assert.ok(!sectionsOf(v.issues).includes('When you open your PR'));
});

test('leftover Fixes #TODO placeholder is flagged even when sections are filled', () => {
  const body = FILLED.replace('Fixes #5', 'Fixes #TODO');
  const v = checkTemplate(body, TEMPLATE);
  assert.equal(v.compliant, false);
  assert.deepEqual(sectionsOf(v.issues), ['Fixes']);
});

test('a missing section is flagged', () => {
  const body = FILLED.replace('### Verification\nRan the e2e tests.\n\n', '');
  const v = checkTemplate(body, TEMPLATE);
  assert.equal(v.compliant, false);
  assert.deepEqual(sectionsOf(v.issues), ['Verification']);
});

test('a section left as only its placeholder comment is flagged as empty', () => {
  const body = FILLED.replace('### Modifications\nFixed the thing.', '### Modifications\n<!-- TODO: Say what changes you made. -->');
  const v = checkTemplate(body, TEMPLATE);
  assert.equal(v.compliant, false);
  assert.deepEqual(sectionsOf(v.issues), ['Modifications']);
});

test('real content plus a leftover placeholder comment is compliant (comment ignored)', () => {
  const body = FILLED.replace('### Documentation\nNot needed — internal change.', '### Documentation\n<!-- TODO: docs. -->\nUpdated the user guide.');
  const v = checkTemplate(body, TEMPLATE);
  assert.equal(v.compliant, true);
});

test('section headers are matched case- and level-insensitively', () => {
  const body = FILLED.replace('### Motivation', '## motivation');
  const v = checkTemplate(body, TEMPLATE);
  assert.equal(v.compliant, true);
});
