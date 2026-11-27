import { test } from 'node:test';
import assert from 'node:assert/strict';
import { CONTRIBUTOR_QUERY, describeAssociation, fetchContributorStats, parseContributorData, renderContributorTable, searchQueries } from '../contributor.ts';
import type { ContributorStats } from '../types.ts';

const data = (overrides: Record<string, unknown> = {}): Record<string, unknown> => ({
  user: { createdAt: '2025-09-23T10:00:00Z' },
  merged: { issueCount: 2 },
  open: { issueCount: 1 },
  closed: { issueCount: 3 },
  issuesOpened: { issueCount: 1 },
  issuesCommented: { issueCount: 4 },
  prsCommented: { issueCount: 5 },
  ...overrides,
});

test('searchQueries scopes every count to the repo and the author', () => {
  const q = searchQueries('argoproj/argo-workflows', 'octo-cat');
  assert.equal(q.merged, 'repo:argoproj/argo-workflows is:pr is:merged author:octo-cat');
  assert.equal(q.open, 'repo:argoproj/argo-workflows is:pr is:open author:octo-cat');
  assert.equal(q.closed, 'repo:argoproj/argo-workflows is:pr is:closed is:unmerged author:octo-cat');
  assert.equal(q.issuesOpened, 'repo:argoproj/argo-workflows is:issue author:octo-cat');
  assert.equal(q.issuesCommented, 'repo:argoproj/argo-workflows is:issue commenter:octo-cat -author:octo-cat');
  assert.equal(q.prsCommented, 'repo:argoproj/argo-workflows is:pr commenter:octo-cat -author:octo-cat');
});

test('searchQueries rejects anything that is not a GitHub login', () => {
  for (const login of ['', 'a b', 'x author:y', '-lead', 'dependabot[bot]', 'a'.repeat(40)]) {
    assert.throws(() => searchQueries('o/r', login), /not a valid GitHub login/, login);
  }
});

test('every search alias in the query has a variable', () => {
  for (const key of Object.keys(searchQueries('o/r', 'u'))) {
    assert.ok(CONTRIBUTOR_QUERY.includes(`${key}: search(query: $${key},`), key);
  }
});

test('parseContributorData reads all counts', () => {
  const stats = parseContributorData(data(), 'FIRST_TIME_CONTRIBUTOR');
  assert.deepEqual(stats, {
    createdAt: '2025-09-23T10:00:00Z',
    association: 'FIRST_TIME_CONTRIBUTOR',
    prs: { merged: 2, open: 1, closed: 3 },
    issues: { opened: 1, commented: 4 },
    othersPrsCommented: 5,
  });
});

test('parseContributorData rejects missing or malformed fields', () => {
  assert.throws(() => parseContributorData(data({ user: null }), 'NONE'), /createdAt/);
  assert.throws(() => parseContributorData(data({ closed: null }), 'NONE'), /closed/);
  assert.throws(() => parseContributorData(data({ merged: { issueCount: -1 } }), 'NONE'), /merged/);
  assert.throws(() => parseContributorData(data({ open: { issueCount: '1' } }), 'NONE'), /open/);
});

test('fetchContributorStats sends the query with the login and search strings', async () => {
  let seen: Record<string, unknown> | null = null;
  const stats = await fetchContributorStats(
    async (query, variables) => {
      assert.equal(query, CONTRIBUTOR_QUERY);
      seen = variables;
      return data();
    },
    'o/r',
    'u',
    'CONTRIBUTOR'
  );
  assert.equal(stats.prs.merged, 2);
  assert.deepEqual(seen, { login: 'u', ...searchQueries('o/r', 'u') });
});

test('renderContributorTable renders the agreed rows in order', () => {
  const stats: ContributorStats = parseContributorData(data(), 'FIRST_TIME_CONTRIBUTOR');
  const table = renderContributorTable('argoproj/argo-workflows', stats, new Date('2026-11-09T00:00:00Z'));
  assert.equal(
    table,
    [
      '| Contributor activity in argoproj/argo-workflows | |',
      '|---|---:|',
      '| PRs (merged / open / closed) | 2 / 1 / 3 |',
      '| Issues (opened / commented on) | 1 / 4 |',
      "| Others' PRs commented on | 5 |",
      '| Account age | 411 days (since 2025-09-23) |',
      '| Association | First-time contributor |',
    ].join('\n')
  );
});

test('renderContributorTable says "1 day" and never a negative age', () => {
  const stats = parseContributorData(data({ user: { createdAt: '2026-11-08T00:00:00Z' } }), 'NONE');
  assert.ok(renderContributorTable('o/r', stats, new Date('2026-11-09T01:00:00Z')).includes('| 1 day (since 2026-11-08) |'));
  assert.ok(renderContributorTable('o/r', stats, new Date('2026-11-01T00:00:00Z')).includes('| 0 days'));
});

test('describeAssociation distinguishes first-time here from first-time on GitHub', () => {
  assert.equal(describeAssociation('FIRST_TIME_CONTRIBUTOR'), 'First-time contributor');
  assert.equal(describeAssociation('FIRST_TIMER'), 'First-time GitHub contributor');
  assert.equal(describeAssociation('SOMETHING_NEW'), 'Unknown');
});
