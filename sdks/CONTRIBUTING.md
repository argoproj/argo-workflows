# Contributing an SDK

Tips;:

* Commit the minimal amount of code possible so other devs don't have to commit lots of files too.
* Commit enough for users to learn how to use it. Use `.gitignore` to exclude files.
* Committed code must be stable, it must not change based on Git tags.
* Provide a [`Makefile`](java/Makefile) with the following:
    * A `generate` target to generate the code using `openapi-generator` into `api` directory.
    * A `build` target to build and test any generated code to make sure it complies.
    * A `publish` target to publish the generated code for use.
* Add a [README.md](java/README.md) to help users get started.