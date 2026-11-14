// Contributor activity table for the sticky comment: how much the PR author
// has already done in this repository, so a maintainer has context at a
// glance. Plain counts only — no thresholds or verdicts.
//
// Everything comes from one GraphQL query: the account's creation date plus
// aliased search counts. The "commented on" figures count distinct threads
// (search's `commenter:` qualifier), not individual comments — paging through
// a user's entire comment history is unbounded and fails on comments in
// private repositories.

import type { ContributorStats } from './types.ts';

// GitHub logins: alphanumerics and single hyphens, at most 39 characters.
// Checked before the login goes anywhere near a search string.
const LOGIN_PATTERN = /^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$/;

const SEARCH_KEYS = ['merged', 'open', 'closed', 'issuesOpened', 'issuesCommented', 'prsCommented'] as const;
type SearchKey = (typeof SEARCH_KEYS)[number];

export const CONTRIBUTOR_QUERY = `
query ContributorActivity(
  $login: String!
  $merged: String!
  $open: String!
  $closed: String!
  $issuesOpened: String!
  $issuesCommented: String!
  $prsCommented: String!
) {
  user(login: $login) { createdAt }
  merged: search(query: $merged, type: ISSUE, first: 1) { issueCount }
  open: search(query: $open, type: ISSUE, first: 1) { issueCount }
  closed: search(query: $closed, type: ISSUE, first: 1) { issueCount }
  issuesOpened: search(query: $issuesOpened, type: ISSUE, first: 1) { issueCount }
  issuesCommented: search(query: $issuesCommented, type: ISSUE, first: 1) { issueCount }
  prsCommented: search(query: $prsCommented, type: ISSUE, first: 1) { issueCount }
}`;

export function searchQueries(repoFullName: string, login: string): Record<SearchKey, string> {
  if (!LOGIN_PATTERN.test(login)) {
    throw new Error(`not a valid GitHub login: ${JSON.stringify(login)}`);
  }
  const repo = `repo:${repoFullName}`;
  return {
    merged: `${repo} is:pr is:merged author:${login}`,
    open: `${repo} is:pr is:open author:${login}`,
    // Merged PRs are closed too; count only the ones closed without merging.
    closed: `${repo} is:pr is:closed is:unmerged author:${login}`,
    issuesOpened: `${repo} is:issue author:${login}`,
    // Threads started by someone else, so these measure engagement beyond
    // the author's own issues and PRs.
    issuesCommented: `${repo} is:issue commenter:${login} -author:${login}`,
    prsCommented: `${repo} is:pr commenter:${login} -author:${login}`,
  };
}

function count(data: Record<string, unknown>, key: SearchKey): number {
  const value = (data[key] as { issueCount?: unknown } | null | undefined)?.issueCount;
  if (typeof value !== 'number' || !Number.isInteger(value) || value < 0) {
    throw new Error(`invalid search count for ${key}`);
  }
  return value;
}

export function parseContributorData(data: Record<string, unknown>, association: string): ContributorStats {
  const createdAt = (data.user as { createdAt?: unknown } | null | undefined)?.createdAt;
  if (typeof createdAt !== 'string' || Number.isNaN(Date.parse(createdAt))) {
    throw new Error('invalid or missing user.createdAt');
  }
  return {
    createdAt,
    association,
    prs: { merged: count(data, 'merged'), open: count(data, 'open'), closed: count(data, 'closed') },
    issues: { opened: count(data, 'issuesOpened'), commented: count(data, 'issuesCommented') },
    othersPrsCommented: count(data, 'prsCommented'),
  };
}

export type GraphQL = (query: string, variables: Record<string, unknown>) => Promise<Record<string, unknown>>;

export async function fetchContributorStats(
  graphql: GraphQL,
  repoFullName: string,
  login: string,
  association: string
): Promise<ContributorStats> {
  const data = await graphql(CONTRIBUTOR_QUERY, { login, ...searchQueries(repoFullName, login) });
  return parseContributorData(data, association);
}

// The PR's author_association, in words. FIRST_TIMER means a first
// contribution anywhere on GitHub; FIRST_TIME_CONTRIBUTOR a first one here.
const ASSOCIATIONS: Record<string, string> = {
  FIRST_TIMER: 'First-time GitHub contributor',
  FIRST_TIME_CONTRIBUTOR: 'First-time contributor',
  CONTRIBUTOR: 'Contributor',
  COLLABORATOR: 'Collaborator',
  MEMBER: 'Organization member',
  OWNER: 'Owner',
  MANNEQUIN: 'Mannequin',
  NONE: 'None',
};

export function describeAssociation(association: string): string {
  return ASSOCIATIONS[association] ?? 'Unknown';
}

const DAY_MS = 24 * 60 * 60 * 1000;

export function renderContributorTable(repoFullName: string, stats: ContributorStats, now: Date): string {
  const created = new Date(stats.createdAt);
  const ageDays = Math.max(0, Math.floor((now.getTime() - created.getTime()) / DAY_MS));
  const since = created.toISOString().slice(0, 10);
  return [
    `| Contributor activity in ${repoFullName} | |`,
    '|---|---:|',
    `| PRs (merged / open / closed) | ${stats.prs.merged} / ${stats.prs.open} / ${stats.prs.closed} |`,
    `| Issues (opened / commented on) | ${stats.issues.opened} / ${stats.issues.commented} |`,
    `| Others' PRs commented on | ${stats.othersPrsCommented} |`,
    `| Account age | ${ageDays} ${ageDays === 1 ? 'day' : 'days'} (since ${since}) |`,
    `| Association | ${describeAssociation(stats.association)} |`,
  ].join('\n');
}
