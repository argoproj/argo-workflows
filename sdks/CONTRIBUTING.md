# Contributing an SDK

Yes please!

* Commit the minimal amount of code possible so other devs don't have to commit lots of files to.
* Commit enough for users to learn how to use it. Use `.gitignore` to exclude files.
* Committed code should be stable, should not change based on Git tags.
* Provide a `Makefile` with the following:
    * A `generate` target to generate the code using `openapi-generator`.
    * A `lint` target to lint the generated code using `prettier` so that is opinionated.
    * A `build` target to build and test any generated code to make sure it complies.
    * A `publish` target to publish the generate code for use.
