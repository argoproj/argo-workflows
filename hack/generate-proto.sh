#!/bin/bash
set -eu -o pipefail

trap 'rm -Rf vendor' EXIT

find pkg -name '*.proto' | while read -r f; do
  newest_output=$(ls -t "$(dirname $f)"/*.pb.go | head -n1)
  swagger_output=$(echo $f | sed 's/.proto/.swagger.json/')
  if [ "$f" -nt "$newest_output" ] || [ ! -e "$swagger_output" ]; then
    echo "running protoc $f"
    [ -e vendor ] || go mod vendor
    protoc \
      -I /usr/local/include \
      -I . \
      -I ./vendor \
      -I ${GOPATH}/src \
      -I ${GOPATH}/pkg/mod/github.com/gogo/protobuf@v1.3.1/gogoproto \
      -I ${GOPATH}/pkg/mod/github.com/grpc-ecosystem/grpc-gateway@v1.12.2/third_party/googleapis \
      --gogofast_out=plugins=grpc:${GOPATH}/src \
      --grpc-gateway_out=logtostderr=true:${GOPATH}/src \
      --swagger_out=logtostderr=true,fqn_for_swagger_name=true:. \
      $f 2>&1 |
      grep -v 'warning: Import .* is unused'
    touch "${f/.proto/.pb.go}"
    touch "${f/.proto/.swagger.json}"
    [ -e ${f/.proto/.pb.gw.go} ] && touch "${f/.proto/.pb.gw.go}"
  else
    echo "skipping protoc $f: no changes"
  fi
done
