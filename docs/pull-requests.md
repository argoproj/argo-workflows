# Pull Requests

Use this checklist when raising a pull request.
The [pull request template](https://github.com/argoproj/argo-workflows/blob/main/.github/pull_request_template.md) mirrors it, and a bot will convert a PR back to draft if the template is not filled in.

## Before You Start

* Features need an associated issue, and should be discussed there before development.
* Changes without either unit or e2e tests are unlikely to be accepted.
* New dependencies must pass the [dependency tests](#dependencies).
* Documentation changes must follow the [docs contribution guide](doc-changes.md).

## Dependencies

Dependencies increase the risk of security issues and have on-going maintenance costs.

A new dependency must pass these tests:

* A strong use case.
* It has an acceptable license (e.g. MIT).
* It is actively maintained.
* It has no security issues.

Example, should we add `fasttemplate`, [view the Snyk report](https://snyk.io/advisor/golang/github.com/valyala/fasttemplate):

| Test                                    | Outcome                             |
|-----------------------------------------|-------------------------------------|
| A strong use case.                      | ❌ Fail. We can use `text/template`. |
| It has an acceptable license (e.g. MIT) | ✅ Pass. MIT license.               |
| It is actively maintained.              | ❌ Fail. Project is inactive.        |
| It has no security issues.              | ✅ Pass. No known security issues.  |

No, we should not add that dependency.

## Commits

Before you commit code and raise a PR, always run:

```bash
make pre-commit -B
```

* [Sign-off](https://probot.github.io/apps/dco) your commits.
* Use [Conventional Commit messages](https://www.conventionalcommits.org/en/v1.0.0/).

For example:

```bash
git commit --signoff -m 'fix: Fixed broken thing'
```

## Feature Description Files

When adding a new feature, you must create a feature description file, which is used to generate the release notes:

```bash
make feature-new
```

This creates a file in the `.features` directory, named after your current branch by default; override the name with `make feature-new FEATURE_FILENAME=my-awesome-feature`.
The file name isn't used by the tooling, it just needs to be unique to avoid collisions on merge.
Edit the file to describe your feature: the issue number is required, and the `Component` field must match one of the components in `hack/featuregen/components.go`.
Validate the file with `make features-validate` (also run by `pre-commit`), and preview its release notes entry with `make features-preview`.
Include the file in your PR.

## Opening the PR

* Create the PR as draft, and mark it "Ready for review" once builds are green.
* The PR title must conform to [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/), as it is used for the release notes.
* Replace `Fixes #TODO` with the issue this PR fixes, or remove the line if it does not fix an issue.
* Fill in every section of the pull request template: Motivation, Modifications, Verification, Documentation, and AI.
* Declare any use of generative AI tools in the `AI` section, following the [Argo project Generative AI policy](https://github.com/argoproj/argoproj/blob/main/community/genai.md); say "None" if no AI was used.
* When changes are requested, address them and then dismiss the review to get it reviewed again.
