#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

CURR_DIR=$(pwd)

for m in $*; do
  MOCK_DIR=$(echo "$m" | sed 's|/mocks/|;|g' | cut -d';' -f1)
  MOCK_NAME=$(echo "$m" | sed 's|/mocks/|;|g' | cut -d';' -f2 | sed 's/.go//g')

  cd "$MOCK_DIR" && mockery -name=$"$MOCK_NAME"
  cd "$CURR_DIR"
done