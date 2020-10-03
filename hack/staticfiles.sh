#!/usr/bin/env bash
set -eu -o pipefail

cd "$(dirname $0)/.."

if [ api/openapi-spec/swagger.json -nt ui/dist/app/index.html ] || [ ui/src/app -nt ui/dist/app/index.html ] ; then
  if [ "${STATIC_FILES:=true}" = true ]; then
    yarn --cwd ui install
    JOBS=max yarn --cwd ui build
  else
    mkdir -p ui/dist/app
    echo "Built without static files" > ui/dist/app/index.html
  fi
else
  echo "skipping UI build: no changes"
fi

if [ ui/dist/app/index.html -nt server/static/files.go ]; then
  staticfiles -o server/static/files.go ui/dist/app
else
  echo "skipping static files: no changes"
fi