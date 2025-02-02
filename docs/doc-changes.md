# Documentation Changes

Docs help our customers understand how to use workflows and fix their own problems.

Doc changes are checked for spelling, broken links, and lint issues by CI. To check locally, run `make docs`.

General guidelines:

* Explain when you would want to use a feature.
* Provide working examples.
    * Prefer to [embed files](https://squidfunk.github.io/mkdocs-material/reference/code-blocks/#embedding-external-files) from the `examples/` directory.
* Format code using back-ticks to avoid it being reported as a spelling error.
* Prefer 1 sentence per line of markdown.
* Internally, refer to "Argo Workflows" as "Argo", unless a disambiguation is needed (such as when integrating with Argo CD).
* Use relative links to other pages or sections
    * For example, `[architecture](architecture.md)`
* When documenting a new feature, add an in-line version annotation
    * For example, `> v3.6 and after`
* Follow the recommendations in the official [Kubernetes Documentation Style Guide](https://kubernetes.io/docs/contribute/style/style-guide/).
    * Particularly useful sections include [Content best practices](https://kubernetes.io/docs/contribute/style/style-guide/#content-best-practices) and [Patterns to avoid](https://kubernetes.io/docs/contribute/style/style-guide/#patterns-to-avoid).
    * **Note**: Argo does not use the same tooling, so the sections on "shortcodes" and "EditorConfig" are not relevant.

## Running Locally

To test/run locally:

```bash
make docs-serve
```

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
