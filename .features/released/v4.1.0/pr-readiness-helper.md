Description: PR readiness helper bot guides contributors through fixing CI failures
Authors: [Alan Clucas](https://github.com/Joibel)
Component: Build and Development
Issues: 16231

A bot now helps contributors get their PRs ready for review.
When CI completes on a PR it maintains a single comment listing the contributor-fixable problems — lint, codegen, UI, build, docs, PR title format, missing feature files, DCO sign-off and an unfilled PR description — each with the command to fix it.
PRs with blocking problems are moved to draft; mark the PR ready for review again once they are fixed.
The comment updates as checks change and shows all-clear when everything contributor-fixable is resolved.
Unit and E2E test results are not covered by the bot.
Maintainers can tune the covered checks and guidance in `.github/pr-readiness/checks.config.json`.
