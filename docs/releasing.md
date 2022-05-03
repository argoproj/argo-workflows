# Release Instructions

## Release

### Cherry-pick Issue

Create a cherry-pick issue to allow the team and community to comment on the release contents.

1. Locate the previous cherry-pick issue
2. Get the hash of the most recent commit still available on the previous issue
3. Generate new issue contents:

    ```bash
    git checkout master # Ensure we are on master
    git log --pretty=format:"%an: %s %h"  [COMMIT_HASH]..HEAD
    ```

4. Create a new issue on GitHub with the title `[VERSION] cherry-pick` (e.g. `v3.0.2 cherry-pick`) and the generated commits
as content.

### Cherry-pick to Release Branch

Once the team and community is satisfied with the commits to be cherry-picked, cherry-pick them into the appropriate
release branch. There should be a single release branch per minor release (e.g. `release-3.0`, `release-3.1`, etc.)

1. Checkout the release branch and cherry-pick commits

    ```bash
    git checkout release-3.0
    git cherry-pick [COMMIT_IDS...]
    ```

2. Hope for few merge conflicts!

    A merge conflict during cherry-picking usually means the commit is based on another commit that should be
    cherry-picked first. In case of a merge conflict, you can undo the cherry-picking by `git cherry-pick --abort` and
    revisit the list of commits to make sure the prior commits are cherry-picked as well.

3. Once done cherry-picking, push the release branch to ensure the branch can build and all tests pass.

### Publish Release

Push a new tag to the release branch. Github Actions will automatically build and publish your release. This takes about
1h. Make sure you check this was successful.
