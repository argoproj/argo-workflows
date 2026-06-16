import { test } from 'node:test';
import assert from 'node:assert/strict';
import { parseAiVerdict } from '../ai.ts';
import { pickStepGuidance } from '../classify.ts';

// --- parseAiVerdict: strict, fail-closed validation of model output ---

test('parseAiVerdict accepts a valid compliant verdict', () => {
  const v = parseAiVerdict('{"compliant": true, "issues": []}');
  assert.deepEqual(v, { compliant: true, issues: [] });
});

test('parseAiVerdict accepts a valid non-compliant verdict with issues', () => {
  const v = parseAiVerdict('{"compliant": false, "issues": [{"section": "Motivation", "problem": "TODO placeholder left in place"}]}');
  assert.equal(v?.compliant, false);
  assert.equal(v?.issues.length, 1);
});

test('parseAiVerdict tolerates a fenced code block around the JSON', () => {
  const v = parseAiVerdict('```json\n{"compliant": true, "issues": []}\n```');
  assert.deepEqual(v, { compliant: true, issues: [] });
});

test('parseAiVerdict fails closed (null) on garbage, wrong types, unknown sections, extra props', () => {
  assert.equal(parseAiVerdict('I think it looks fine!'), null);
  assert.equal(parseAiVerdict('{"compliant": "yes", "issues": []}'), null);
  assert.equal(parseAiVerdict('{"compliant": true}'), null);
  assert.equal(parseAiVerdict('{"compliant": false, "issues": [{"section": "Banana", "problem": "x"}]}'), null);
  assert.equal(parseAiVerdict('{"compliant": false, "issues": [{"section": "AI", "problem": "x", "extra": 1}]}'), null);
  assert.equal(parseAiVerdict('{"compliant": true, "issues": [], "extra": true}'), null);
  assert.equal(parseAiVerdict(''), null);
  assert.equal(parseAiVerdict(undefined), null);
});

test('parseAiVerdict caps issues at 6 and problem length at 200', () => {
  const many = JSON.stringify({ compliant: false, issues: Array.from({ length: 9 }, () => ({ section: 'Other', problem: 'p' })) });
  assert.equal(parseAiVerdict(many), null);
  const long = JSON.stringify({ compliant: false, issues: [{ section: 'Other', problem: 'x'.repeat(201) }] });
  assert.equal(parseAiVerdict(long), null);
});

// --- pickStepGuidance: choose per-step guidance for the features check ---

const signal = {
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
  assert.equal(pickStepGuidance(signal, steps), 'run make feature-new');
});

test('pickStepGuidance falls back to generic guidance when no mapped step failed', () => {
  assert.equal(pickStepGuidance(signal, [{ name: 'something else', conclusion: 'failure' }]), 'generic feature guidance');
  assert.equal(pickStepGuidance(signal, null), 'generic feature guidance');
  assert.equal(pickStepGuidance({ guidance: 'g', stepGuidance: null }, []), 'g');
});
