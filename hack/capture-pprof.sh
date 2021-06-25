#!/usr/bin/env bash
set -eu -o pipefail

echo "  https://blog.golang.org/pprof"

cd $(dirname $0)/..

n=$(date +%s)

go tool pprof -web http://localhost:6060/debug/pprof/allocs
go tool pprof -web http://localhost:6060/debug/pprof/heap
go tool pprof -web http://localhost:6060/debug/pprof/profile