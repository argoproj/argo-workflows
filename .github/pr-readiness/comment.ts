// Renders the sticky PR-readiness comment. One comment per PR, identified by
// MARKER, edited in place.

import type { CommentVariant, TemplateIssue } from './types.ts';

export const MARKER = '<!-- pr-readiness-bot -->';

// Must match the label that exists in the repo (Settings → Labels):
// "problem/bot-not-ready — Readiness bot declares this as not ready, see
// comment by bot for why".
export const NOT_READY_LABEL = 'problem/bot-not-ready';

const FOOTER =
  '\n---\n<sub>🤖 Automated PR-readiness helper — it re-checks each time CI finishes. ' +
  'Unit/E2E test results are <b>not</b> covered here. ' +
  'Questions? See [the contributing guide](https://github.com/argoproj/argo-workflows/blob/main/docs/CONTRIBUTING.md) or ask a maintainer.</sub>';

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
  labeled: boolean;
}

export function renderComment({ variant, failures, templateIssues, labeled }: RenderArgs): string {
  const head = [MARKER, ''];

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

  if (labeled) {
    lines.push(
      '',
      '> [!NOTE]',
      `> This PR carries the \`${NOT_READY_LABEL}\` label while the items above are addressed. It is removed automatically once everything passes.`
    );
  }

  lines.push(FOOTER);
  return lines.join('\n');
}
