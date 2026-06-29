// Shared domain types for the PR Readiness Helper. Type-only module: it
// strips to nothing at runtime, so always import from it with `import type`.

export type SignalState = 'pending' | 'failure' | 'success' | 'not-applicable';

export interface SignalMatch {
  check: string;
  app?: string;
}

export interface SignalConfig {
  id: string;
  match: SignalMatch;
  title: string;
  guidance: string;
  stepGuidance?: Record<string, string> | null;
}

export interface Config {
  signals: SignalConfig[];
  ignoreChecks: string[];
  coveredApps: string[];
}

export interface CheckRun {
  name: string;
  status: string;
  conclusion: string | null;
  html_url: string;
  id?: number;
  app?: { slug: string } | null;
}

export interface Signal {
  id: string;
  title: string;
  guidance: string;
  stepGuidance: Record<string, string> | null;
  state: SignalState;
  url: string | null;
}

export interface JobStep {
  name: string;
  conclusion: string | null;
}

export interface TemplateIssue {
  section: string;
  problem: string;
}

export interface TemplateVerdict {
  compliant: boolean;
  issues: TemplateIssue[];
}

export interface State {
  v: number;
  failing: string[];
  draftedSha?: string | null;
}

export interface PrRef {
  draft: boolean;
  headSha: string;
}

export type CommentVariant = 'issues' | 'waiting' | 'allclear';

export interface Decision {
  variant: CommentVariant | null;
  shouldComment: boolean;
  shouldDraft: boolean;
  failing: string[];
  templateBlocking: boolean;
}

export interface GitHubUser {
  login: string;
  type: string;
}
