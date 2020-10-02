#!/usr/bin/env bash
set -eu -o pipefail

cd "$(dirname $0)/../.."

if [ ui/dist/app -nt ui/dist/app/index.html ]; then
  echo "changes since last build"
  if [ "${STATIC_FILES:=true}" = true ]; then
    yarn --cwd ui install
    JOBS=max yarn --cwd ui build
  else
    mkdir -p ui/dist/app
    echo "Built without static files" > ui/dist/app/index.html
  fi
else
  echo "no changes since last build"
fi

staticfiles -o "$(dirname $0)/files.go" ui/dist/app