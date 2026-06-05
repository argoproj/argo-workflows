'use strict';
// Strict, fail-closed parsing of the model's template-compliance verdict.
// The shape mirrors ai-schema.json; validation is hand-rolled so the bot has
// no runtime dependencies. Any deviation => null => the AI layer is ignored.

const SECTIONS = new Set(['Fixes', 'Motivation', 'Modifications', 'Verification', 'Documentation', 'AI', 'Other']);
const MAX_ISSUES = 6;
const MAX_PROBLEM_LENGTH = 200;

function parseAiVerdict(text) {
  if (typeof text !== 'string' || text.trim() === '') {
    return null;
  }
  // Models sometimes wrap JSON in a fenced code block despite instructions.
  const unfenced = text.trim().replace(/^```(?:json)?\s*/, '').replace(/\s*```$/, '');
  let obj;
  try {
    obj = JSON.parse(unfenced);
  } catch {
    return null;
  }
  if (typeof obj !== 'object' || obj === null || Array.isArray(obj)) {
    return null;
  }
  const keys = Object.keys(obj).sort();
  if (keys.length !== 2 || keys[0] !== 'compliant' || keys[1] !== 'issues') {
    return null;
  }
  if (typeof obj.compliant !== 'boolean' || !Array.isArray(obj.issues) || obj.issues.length > MAX_ISSUES) {
    return null;
  }
  for (const issue of obj.issues) {
    if (typeof issue !== 'object' || issue === null || Array.isArray(issue)) {
      return null;
    }
    const issueKeys = Object.keys(issue).sort();
    if (issueKeys.length !== 2 || issueKeys[0] !== 'problem' || issueKeys[1] !== 'section') {
      return null;
    }
    if (!SECTIONS.has(issue.section) || typeof issue.problem !== 'string' || issue.problem.length > MAX_PROBLEM_LENGTH) {
      return null;
    }
  }
  return obj;
}

module.exports = { parseAiVerdict };
