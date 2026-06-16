// Sanitizers for untrusted text (PR bodies, AI output) before it is embedded
// in a comment the bot posts. See README.md "Security".

// Wrap @mentions in backticks so GitHub renders them inert (no notification).
// Skips emails (preceded by a word char) and already-escaped `@name`.
export function neutralizeMentions(text: string): string {
  return text.replace(/(^|[^`\w])@([\w-]+(?:\/[\w-]+)?)/g, '$1`@$2`');
}

// Remove "fixes #N"-style issue-closing references so bot/AI text can never
// link or auto-close an issue.
export function redactClosingKeywords(text: string): string {
  return text.replace(/\b(?:fix(?:e[sd])?|close[sd]?|resolve[sd]?)\s+#\d+/gi, '[issue ref removed]');
}

export function sanitizeAiText(text: string, maxLen: number): string {
  let out = redactClosingKeywords(neutralizeMentions(text));
  if (out.length > maxLen) {
    out = out.slice(0, maxLen) + '…';
  }
  return out;
}
