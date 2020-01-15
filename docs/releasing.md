# Release Instructions

1. Prepare release:

```
make prepare-release VERSION=v2.5.0
```

2. Release:

```
make release VERSION=$VERSION
```

3. If stable:

```
git tag stable
git push stable stable
```

4. Update Homebrew.

5. Draft GitHub release with the content from CHANGELOG.md, and CLI binaries produced in the `dist` directory

* https://github.com/argoproj/argo/releases/new
