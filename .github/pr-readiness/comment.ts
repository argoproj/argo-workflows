// Renders the sticky PR-readiness comment. One comment per PR, identified by
// MARKER, edited in place. A hidden state blob carries data between runs.

import type { CommentVariant, State, TemplateIssue } from './types.ts';

export const MARKER = '<!-- pr-readiness-bot -->';

const FOOTER =
  '\n---\n<sub>🤖 Automated PR-readiness helper — it re-checks each time CI finishes. ' +
  'Unit/E2E test results are <b>not</b> covered here. ' +
  'Questions? See [the contributing guide](https://github.com/argoproj/argo-workflows/blob/main/docs/CONTRIBUTING.md) or ask a maintainer.</sub>';

function stateLine(state: State): string {
  return `<!-- state: ${JSON.stringify(state)} -->`;
}

interface FailureItem {
  title: string;
  guidance: string;
  url: string | null;
  id?: string;
}

interface RenderArgs {
  variant: CommentVariant | null;
  failures: ReadonlyArray<FailureItem>;
  templateIssues: TemplateIssue[] | null;
  // The PR is a draft the bot imposed and has not lifted yet.
  holdingDraft: boolean;
  // The bot drafted this PR and could not lift the draft itself, so the
  // all-clear must ask the contributor to do it rather than tell them to wait
  // for a maintainer who cannot see a draft PR.
  needsReadyForReview: boolean;
  state: State;
}

export function renderComment({ variant, failures, templateIssues, holdingDraft, needsReadyForReview, state }: RenderArgs): string {
  const head = [MARKER, stateLine(state), ''];

  // Rendered on every pass while the bot holds the draft, not just the one
  // that imposed it: the comment is edited in place, so whatever it says last
  // is all the contributor sees. It must always explain why the PR is a draft
  // and that they are not the ones who have to undo it.
  const draftNote = holdingDraft
    ? [
        '',
        '> [!NOTE]',
        '> This PR was moved to **draft** by this helper. It will be marked',
        '> **Ready for review** again automatically once everything here is green.',
      ]
    : [];

  if (variant === 'allclear') {
    return head
      .concat([
        '#### ✅ PR readiness: all clear',
        '',
        needsReadyForReview
          ? 'All contributor-fixable checks are passing. This PR is still a **draft** — mark it **Ready for review** so a maintainer picks it up.'
          : 'All contributor-fixable checks are passing. A maintainer will take it from here — thanks!',
        FOOTER,
      ])
      .join('\n');
  }

  if (variant === 'waiting') {
    return head
      .concat(['#### ⏳ PR readiness', '', 'The earlier issues are resolved — waiting for the remaining checks to finish…'])
      .concat(draftNote)
      .concat([FOOTER])
      .join('\n');
  }

  // variant === 'issues'
  const lines = head.concat([
    '#### 👋 PR readiness check',
    '',
    'Thanks for your contribution! A few automated checks need attention before a maintainer reviews — these are all things you can fix yourself:',
    '',
  ]);

  for (const f of failures) {
    lines.push(`- **${f.title}** — ${f.guidance} ([log](${f.url}))`);
  }

  if (templateIssues && templateIssues.length > 0) {
    lines.push(
      '',
      '<details>',
      '<summary><b>PR description / template</b></summary>',
      '',
      'The PR description does not appear to follow [the template](https://github.com/argoproj/argo-workflows/blob/main/.github/pull_request_template.md):',
      ''
    );
    for (const issue of templateIssues) {
      lines.push(`- **${issue.section}**: ${issue.problem}`);
    }
    lines.push('', '_(A maintainer may waive this.)_', '</details>');
  }

  lines.push(...draftNote);
  lines.push(FOOTER);
  return lines.join('\n');
}

// Returns the state object embedded in a bot comment, or null if the comment
// is not ours / has no parsable state.
export function parseState(body: string): State | null {
  // includes, not startsWith: the sticky-comment action injects its own
  // hidden header into the body it posts.
  if (typeof body !== 'string' || !body.includes(MARKER)) {
    return null;
  }
  const m = body.match(/<!-- state: (.*?) -->/);
  if (!m) {
    return null;
  }
  try {
    return JSON.parse(m[1]) as State;
  } catch {
    return null;
  }
}
