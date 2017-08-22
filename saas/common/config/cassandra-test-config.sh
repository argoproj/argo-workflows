#!/bin/bash

# Pass in the argument that is the directory of the cassandra configurations
if [ "$#" -ne 1 ]; then
    echo "You must enter the directory of the cassandra configurations"
    exit 1
fi

set -e
sed -i -e 's/#MAX_HEAP_SIZE="4G"/MAX_HEAP_SIZE="256M"/g' $1/cassandra-env.sh
sed -i -e 's/#HEAP_NEWSIZE="800M"/HEAP_NEWSIZE="12M"/g' $1/cassandra-env.sh

sed -i -e 's/rpc_server_type: sync/rpc_server_type: hsha/g' $1/cassandra.yaml

sed -i -e 's/concurrent_reads: 32/concurrent_reads: 4/g' $1/cassandra.yaml
sed -i -e 's/concurrent_writes: 32/concurrent_writes: 2/g' $1/cassandra.yaml
sed -i -e 's/concurrent_counter_writes: 32/concurrent_counter_writes: 2/g' $1/cassandra.yaml
sed -i -e 's/concurrent_materialized_view_writes: 32/concurrent_materialized_view_writes: 4/g' $1/cassandra.yaml

sed -i -e 's/# rpc_max_threads: 2048/rpc_max_threads: 2/g' $1/cassandra.yaml
sed -i -e 's/# rpc_min_threads: 16/rpc_min_threads: 1/g' $1/cassandra.yaml

sed -i -e 's/#concurrent_compactors: 1/concurrent_compactors: 1/g' $1/cassandra.yaml
sed -i -e 's/compaction_throughput_mb_per_sec: 16/compaction_throughput_mb_per_sec: 0/g' $1/cassandra.yaml

sed -i -e 's/key_cache_size_in_mb:/key_cache_size_in_mb: 1/g' $1/cassandra.yaml
sed -i -e 's/counter_cache_size_in_mb:/counter_cache_size_in_mb: 1/g' $1/cassandra.yaml

sed -i -e 's/commitlog_segment_size_in_mb: 32/commitlog_segment_size_in_mb: 2/g' $1/cassandra.yaml

sed -i -e 's/# file_cache_size_in_mb: 512/file_cache_size_in_mb: 10/g' $1/cassandra.yaml
sed -i -e 's/auto_snapshot: true/auto_snapshot: false/g' $1/cassandra.yaml
set +e

