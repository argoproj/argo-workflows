# Documentation Changes

Docs help our customers understand how to use workflows and fix their own problems.

Doc changes are checked for spelling, broken links, and lint issues by CI. To check locally run `make docs`.

* Explain when you would want to use a feature.
* Provide working examples.
* Use simple short sentences and avoid jargon.
* Format code using back-ticks to avoid it being reported spelling error.
* Avoid use title-case mid-sentence. E.g. instead of "the Workflow", write "the workflow".
* Headings should be title-case. E.g. instead of "and", write "And".

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
