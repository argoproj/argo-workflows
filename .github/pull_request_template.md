<!-- markdownlint-disable MD041 -- this is rendered within existing HTML, so allow starting without an H1 -->

See the [pull request guide](https://argo-workflows.readthedocs.io/en/latest/pull-requests/) for details on each item.

- [ ] Ran `make pre-commit -B`
- [ ] Signed-off commits with [Conventional Commit](https://www.conventionalcommits.org/en/v1.0.0/) messages
- [ ] PR title is a conventional commit message (it becomes the release notes entry)
- [ ] Unit or e2e tests cover the change
- [ ] For features: an associated issue and a feature description file (`make feature-new`)
- [ ] Opened as draft; will mark "Ready for review" once builds are green

<!-- Does this PR fix an issue -->

Fixes #TODO

### Motivation

<!-- TODO: Say why you made your changes. -->

### Modifications

<!-- TODO: Say what changes you made. -->
<!-- TODO: Attach screenshots if you changed the UI. -->

### Verification

<!-- TODO: Say how you tested your changes. -->

### Documentation

<!-- TODO: Say how you have updated the documentation or explain why this isn't needed here -->
<!-- Required for features: Explain how the user will discover this feature through documentation and examples -->

### AI

<!-- TODO: Declare any use of generative AI tools (for example ChatGPT) in preparing this PR, including code, tests, docs, or commit messages. Say "None" if no AI was used. -->
<!-- See the Argo project's Generative AI policy: https://github.com/argoproj/argoproj/blob/main/community/genai.md -->
