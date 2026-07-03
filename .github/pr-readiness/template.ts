// Deterministic PR-description / template compliance check. Replaces the
// earlier GitHub Models check: it needs no model, no network and no special
// permissions, so it works regardless of org/enterprise policy. It catches
// the common "didn't fill it in" cases (empty body, untouched placeholders,
// missing or empty sections); it deliberately does not judge prose quality.

import type { TemplateVerdict } from './types.ts';

// Strip HTML comments (including multi-line) — they are the template's
// instructional placeholders and never count as real content.
function stripComments(text: string): string {
  return text.replace(/<!--[\s\S]*?-->/g, '');
}

// The required section names are derived from the template itself (its `##`+
// headers, ignoring any inside comment blocks), so the check follows the
// template if it changes.
function sectionHeaders(text: string): string[] {
  return [...stripComments(text).matchAll(/^#{2,6}\s+(.+?)\s*$/gm)].map((m) => m[1].trim());
}

// Split a description into a map of lower-cased section name -> body text.
function splitSections(text: string): Map<string, string> {
  const sections = new Map<string, string>();
  let current: string | null = null;
  let buf: string[] = [];
  const flush = (): void => {
    if (current !== null) {
      sections.set(current.toLowerCase(), buf.join('\n'));
    }
  };
  for (const line of text.split('\n')) {
    const header = line.match(/^#{2,6}\s+(.+?)\s*$/);
    if (header) {
      flush();
      current = header[1].trim();
      buf = [];
    } else if (current !== null) {
      buf.push(line);
    }
  }
  flush();
  return sections;
}

export function checkTemplate(body: string, templateText: string): TemplateVerdict {
  const raw = (body || '').trim();
  if (raw === '') {
    return {
      compliant: false,
      issues: [{ section: 'Description', problem: 'The PR description is empty — please fill in the pull request template.' }],
    };
  }

  const issues: TemplateVerdict['issues'] = [];

  // The `Fixes #TODO` placeholder must be replaced with a real issue number
  // or removed (not every PR fixes an issue).
  if (/Fixes #TODO/i.test(stripComments(body))) {
    issues.push({
      section: 'Fixes',
      problem: 'Replace `Fixes #TODO` with the issue this fixes (e.g. `Fixes #1234`), or remove the line if this PR does not fix an issue.',
    });
  }

  const required = sectionHeaders(templateText);
  const present = splitSections(stripComments(body));
  for (const name of required) {
    const content = present.get(name.toLowerCase());
    if (content === undefined) {
      issues.push({ section: name, problem: `The "${name}" section is missing — please keep it and fill it in.` });
    } else if (content.trim() === '') {
      issues.push({ section: name, problem: `The "${name}" section is empty or still only contains the template placeholder.` });
    }
  }

  return { compliant: issues.length === 0, issues };
}
