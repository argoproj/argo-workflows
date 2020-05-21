#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source $(dirname $0)/library.sh

header "updating mock files"

ensure_vendor

cd ${REPO_ROOT}
if [ ! -e "${GOPATH}/bin/mockery" ]; then
  ./hack/recurl.sh dist/mockery.tar.gz https://github.com/vektra/mockery/releases/download/v1.1.1/mockery_1.1.1_$(uname -s)_$(uname -m).tar.gz
  tar zxvf dist/mockery.tar.gz mockery
  chmod +x mockery
  mkdir -p ${GOPATH}/bin
  mv mockery ${GOPATH}/bin/mockery
fi

MOCKERY_CMD="${GOPATH}/bin/mockery"
${MOCKERY_CMD} -version

MOCK_FILES=$(find persist workflow -maxdepth 4 -not -path '/vendor/*' -not -path './ui/*' -path '*/mocks/*' -type f -name '*.go')

for m in ${MOCK_FILES}; do
  echo $m
  MOCK_DIR=$(echo "$m" | sed 's|/mocks/|;|g' | cut -d';' -f1)
  MOCK_NAME=$(echo "$m" | sed 's|/mocks/|;|g' | cut -d';' -f2 | sed 's/.go//g')

  cd "$MOCK_DIR" && ${MOCKERY_CMD} -name=$"$MOCK_NAME"
  cd ${REPO_ROOT}
done
