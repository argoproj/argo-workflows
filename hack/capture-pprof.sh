#!/usr/bin/env bash
set -eu -o pipefail

echo "  https://blog.golang.org/pprof"

mkdir -p $(dirname $0)/../pprof
cd $(dirname $0)/../pprof

n=$(date +%s)

go tool pprof -png -output heap-$n.png http://localhost:6060/debug/pprof/heap
go tool pprof -png -output allocs-$n.png http://localhost:6060/debug/pprof/allocs
go tool pprof -png -output block-$n.png http://localhost:6060/debug/pprof/block
go tool pprof -png -output mutex-$n.png http://localhost:6060/debug/pprof/mutex
go tool pprof -png -output profile-$n.png http://localhost:6060/debug/pprof/profile?seconds=30