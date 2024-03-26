#!/bin/bash
set -euo pipefail

target="docs/README.md"
cp README.md $target

replaceTarget() {
  # cross-platform for Linux and Mac, based off https://unix.stackexchange.com/a/381201/152866
  sed -i.bak -e "$1" "$target" && rm $target.bak
}

# replace absolute links with relative links
replaceTarget 's/(https:\/\/argo-workflows\.readthedocs\.io\/en\/latest\/\(.*\)\/)/(\1\.md)/'
replaceTarget 's/walk-through\.md/walk-through\/index\.md/' # index routes need special handling
# adjust existing relative links
replaceTarget 's/(docs\//(/' # remove `docs/` prefix
replaceTarget 's/(USERS\.md/(https:\/\/github\.com\/argoproj\/argo-workflows\/blob\/main\/USERS\.md/' # replace non-docs link with an absolute link
replaceTarget 's/SECURITY\.md\](SECURITY/Security\](security/' # case-sensitive -- the file is docs/security.md vs. SECURITY.md. also remove the .md from a docs link

# change text for docs self-link
replaceTarget 's/.*View the docs.*/You'\''re here!/'
