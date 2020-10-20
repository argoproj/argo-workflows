## Committer Checklist

* [ ] Either (a) I've created an [enhancement proposal](https://github.com/argoproj/argo/issues/new/choose) and discussed it with the community, (b) this is a bug fix, or (c) this is a chore.
* [ ] The title of the PR is (a) [conventional](https://www.conventionalcommits.org/en/v1.0.0/), (b) states what changed, and (c) suffixes the related issues number. E.g. `"fix(controller): Updates such and such. Fixes #1234"`.  
* [ ] My organization is added to [USERS.md](https://github.com/argoproj/argo/blob/master/USERS.md).
* [ ] I've signed the CLA.
* [ ] I have written unit and/or e2e tests for my change. 
* [ ] My builds are green. Try syncing with master if they are not. 

## Approver Checklist

* [ ] Includes tests.
* [ ] Correct v3 package name.
* [ ] Is based on `release-2.12` for a v2 bug fix, `master` for anything v3.