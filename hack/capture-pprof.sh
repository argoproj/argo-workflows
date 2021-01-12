#!/usr/bin/env bash
set -eu -o pipefail

echo "  https://blog.golang.org/pprof"

cd $(dirname $0)/..

n=$(date +%s)

go tool pprof -png -output dist/heap-$n.png http://localhost:6060/debug/pprof/heap
go tool pprof -png -output dist/allocs-$n.png http://localhost:6060/debug/pprof/allocs
go tool pprof -png -output dist/block-$n.png http://localhost:6060/debug/pprof/block
go tool pprof -png -output dist/mutex-$n.png http://localhost:6060/debug/pprof/mutex
go tool pprof -png -output dist/profile-$n.png http://localhost:6060/debug/pprof/profile?seconds=30