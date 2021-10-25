# Contributing an SDK

Make it contributor friendly:

* Make it fast, because engineers will have to generate SDKs for every PR.
* Make it dependency free, engineers will not be a be able to install anything. You can use Docker.
* Generate the minimal amount of code possible, so other engineers don't have to commit lots of files too.
* Provide a [`Makefile`](java/Makefile) with the following:
    * A `generate` target to generate the code using `openapi-generator` into `client` directory.
    * A `publish` target to publish the generated code for use.
* Committed code must be stable, it must not change based on Git tags.

Make it user friendly:

* Commit enough for users to learn how to use it. Use `.gitignore` to exclude files.
* Add a [README.md](java/README.md) to help users get started.