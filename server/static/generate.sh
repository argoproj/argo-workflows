#!/usr/bin/env bash
set -eu -o pipefail

cd "$(dirname $0)/../.."

if [ "${STATIC_FILES:=true}" = true ]; then
  yarn --cwd ui build
	JOBS=max yarn --cwd ui build
else
  mkdir -p ui/dist/app
	echo "Built without static files" > ui/dist/app/index.html
fi

staticfiles -o files.go ../../ui/dist/app