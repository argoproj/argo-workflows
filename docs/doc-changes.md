# Documentation Changes

Docs help our customers understand how to use workflows and fix their own problems.

General guidelines:

* Explain when you would want to use a feature.
* Provide working examples.
* Format code using back-ticks to avoid it being reported as a spelling error.
* Prefer 1 sentence per line of markdown
* Follow the recommendations in the official [Kubernetes Documentation Style Guide](https://kubernetes.io/docs/contribute/style/style-guide/).
    * Particularly useful sections include [Content best practices](https://kubernetes.io/docs/contribute/style/style-guide/#content-best-practices) and [Patterns to avoid](https://kubernetes.io/docs/contribute/style/style-guide/#patterns-to-avoid).
    * **Note**: Argo does not use the same tooling, so the sections on "shortcodes" and "EditorConfig" are not relevant.

## Workflow

### Running Locally

To build the docs and start a server at <http://localhost:8000/> where you can preview your changes:

```bash
make docs-serve
```

This command also checks the docs for spelling, broken links, and lint issues.

### Entering a PR

See [the pull request template](https://github.com/argoproj/argo-workflows/blob/main/.github/pull_request_template.md).

On entering a PR, our CI will run the same checks as `make docs-serve`, and fail the build if any issues are found.

Additionally, your PR will be published to a temporary URL, which you can access by clicking on the "Details" link next to the `docs/readthedocs.org:argo-workflows` check.
This can can be used to preview your changes and do a [visual diff](https://docs.readthedocs.com/platform/stable/visual-diff.html).

## Tips

Use a service like [Grammarly](https://www.grammarly.com) to check your grammar.

Having your computer read text out loud is a way to catch problems, e.g.:

* Word substitutions (i.e. the wrong word is used, but spelled.
correctly).
* Sentences that do not read correctly will sound wrong.

On Mac, to set-up:

* Go to `System Preferences / Accessibility / Spoken Content`.
* Choose a System Voice (I like `Siri Voice 1`).
* Enable `Speak selection`.

To hear text, select the text you want to hear, then press option+escape.
