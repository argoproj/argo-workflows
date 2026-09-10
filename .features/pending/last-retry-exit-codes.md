Description: Add lastRetry.exitCodes variable exposing all previous attempt exit codes
Authors: [Liron Shabtai](https://github.com/lirons-legit)
Component: General
Issues: 12849

`lastRetry.exitCodes` is a comma-separated list of the exit codes of all previous retry attempts, oldest first, and is empty on the first attempt.
Unlike `lastRetry.exitCode`, which reports only the immediately previous attempt, it lets an expression accumulate a resource across retries conditionally.
For example, a `podSpecPatch` memory request of `{{= 2 + len(filter(split(lastRetry.exitCodes, ','), {# == '137'})) }}Gi` grows a pod's memory once per previous OOM (exit code 137) and holds it across non-OOM (eviction) retries, which the previous single-attempt variable could not express.
