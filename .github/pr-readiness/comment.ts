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
  drafted: boolean;
  state: State;
}

export function renderComment({ variant, failures, templateIssues, drafted, state }: RenderArgs): string {
  const head = [MARKER, stateLine(state), ''];

  if (variant === 'allclear') {
    return head
      .concat([
        '#### ✅ PR readiness: all clear',
        '',
        'All contributor-fixable checks are passing. A maintainer will take it from here — thanks!',
        FOOTER,
      ])
      .join('\n');
  }

  if (variant === 'waiting') {
    return head
      .concat([
        '#### ⏳ PR readiness',
        '',
        'The earlier issues are resolved — waiting for the remaining checks to finish…',
        FOOTER,
      ])
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

  if (drafted) {
    lines.push(
      '',
      '> [!NOTE]',
      '> This PR has been moved to **draft** while the items above are addressed. Mark it **Ready for review** once they are fixed.'
    );
  }

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
