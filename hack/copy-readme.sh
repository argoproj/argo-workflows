#!/bin/bash
set -euo pipefail

cp README.md docs/README.md

# replace absolute links with relative links
sed -i '' 's/(https:\/\/argo-workflows\.readthedocs\.io\/en\/latest\/\(.*\)\/)/(\1\.md)/' docs/README.md
sed -i '' 's/walk-through\.md/walk-through\/index\.md/' docs/README.md # index routes need special handling
# adjust existing relative links
sed -i '' 's/(docs\//(/' docs/README.md # remove `docs/` prefix
sed -i '' 's/(USERS\.md/(https:\/\/github\.com\/argoproj\/argo-workflows\/blob\/main\/USERS\.md/' docs/README.md # replace non-docs link with an absolute link

# change text for docs self-link
sed -i '' 's/.*View the docs.*/You'\''re here!/' docs/README.md
