#!/bin/bash

# Copyright 2014 The Kubernetes Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

service cassandra stop
sed -e "\|/var/lib/cassandra|s|/var/lib/cassandra|$CASSANDRA_HOME|" /etc/cassandra/cassandra.yaml | sed -e 's/- class_name: .*SeedProvider$/- class_name: io.k8s.cassandra.KubernetesSeedProvider/'> /tmp/ca.yaml
cp /tmp/ca.yaml /etc/cassandra/cassandra.yaml
export CLASSPATH=/ax/axdb/kubernetes-cassandra.jar

set -e
CFG=/etc/cassandra/cassandra.yaml
ENVCFG=/etc/cassandra/cassandra-env.sh
CASSANDRA_RPC_ADDRESS="${CASSANDRA_RPC_ADDRESS:-0.0.0.0}"
CASSANDRA_NUM_TOKENS="${CASSANDRA_NUM_TOKENS:-256}"
CASSANDRA_CLUSTER_NAME="${CASSANDRA_CLUSTER_NAME:='Test Cluster'}"
CASSANDRA_SERVICE="${CASSANDRA_SERVICE:=axdb}"
CASSANDRA_LISTEN_ADDRESS=${POD_IP}
CASSANDRA_BROADCAST_ADDRESS=${POD_IP}
CASSANDRA_BROADCAST_RPC_ADDRESS=${POD_IP}
CASSANDRA_RANGE_REQUEST_TIMEOUT_IN_MS="${CASSANDRA_RANGE_REQUEST_TIMEOUT_IN_MS:-30000}"
CASSANDRA_READ_REQUEST_TIMEOUT_IN_MS="${CASSANDRA_READ_REQUEST_TIMEOUT_IN_MS:-30000}"
CASSANDRA_COUNTER_WRITE_REQUEST_TIMEOUT_IN_MS="${CASSANDRA_COUNTER_WRITE_REQUEST_TIMEOUT_IN_MS:-10000}"
CASSANDRA_CAS_CONTENTION_TIMEOUT_IN_MS="${CASSANDRA_CAS_CONTENTION_TIMEOUT_IN_MS:-5000}"
CASSANDRA_WRITE_REQUEST_TIMEOUT_IN_MS="${CASSANDRA_WRITE_REQUEST_TIMEOUT_IN_MS:-10000}"
CASSANDRA_REQUEST_TIMEOUT_IN_MS="${CASSANDRA_REQUEST_TIMEOUT_IN_MS:-10000}"
CASSANDRA_TOMBSTONE_FAILURE_THRESHOLD="${CASSANDRA_TOMBSTONE_FAILURE_THRESHOLD:-1000000}"
: ${NUMBER_OF_NODES:=1}

for yaml in \
  broadcast_address \
	broadcast_rpc_address \
	cluster_name \
	listen_address \
	num_tokens \
	rpc_address \
	range_request_timeout_in_ms \
	read_request_timeout_in_ms \
	counter_write_request_timeout_in_ms \
	cas_contention_timeout_in_ms \
	write_request_timeout_in_ms \
	request_timeout_in_ms \
	tombstone_failure_threshold \
; do
  var="CASSANDRA_${yaml^^}"
	val="${!var}"
	if [ "$val" ]; then
		sed -ri 's/^(# )?('"$yaml"':).*/\2 '"$val"'/' "$CFG"
	fi
done

# to uncomment hints_directory
sed -ie 's/^# \(hints_directory:.*\)/\1/' "$CFG"

# Eventual do snitch $DC && $RACK?
#if [[ $SNITCH ]]; then
#  sed -i -e "s/endpoint_snitch: SimpleSnitch/endpoint_snitch: $SNITCH/" $CONFIG/cassandra.yaml
#fi
#if [[ $DC && $RACK ]]; then
#  echo "dc=$DC" > $CONFIG/cassandra-rackdc.properties
#  echo "rack=$RACK" >> $CONFIG/cassandra-rackdc.properties
#fi

# if cassandra memory limit is specified in environment variable; override the system calculated value
if [ -n "$CASSANDRA_MEM_LIMIT" ]; then
  [[ -n $MEM_MULT ]] || ( echo "Must set MEM_MULT" ; exit 1)
  mem_size=$(printf "%.0f" $(bc -l <<< "$CASSANDRA_MEM_LIMIT * $MEM_MULT"))
  echo mem_size ${mem_size}
  export mem_size=${mem_size}
  echo 'JVM_OPTS="$JVM_OPTS -Xmx${mem_size}M -Xms${mem_size}M"' >> $ENVCFG
fi

echo "Starting Cassandra on $POD_IP"
echo CASSANDRA_RPC_ADDRESS ${CASSANDRA_RPC_ADDRESS}
echo CASSANDRA_NUM_TOKENS ${CASSANDRA_NUM_TOKENS}
echo CASSANDRA_CLUSTER_NAME ${CASSANDRA_CLUSTER_NAME}
echo CASSANDRA_LISTEN_ADDRESS ${POD_IP}
echo CASSANDRA_BROADCAST_ADDRESS ${POD_IP}
echo CASSANDRA_BROADCAST_RPC_ADDRESS ${POD_IP}

MODE="Standlone"
if [ "$NUMBER_OF_NODES" -gt 1 ]; then
MODE="Cluster"
fi

echo NUMBER_OF_NODES: $NUMBER_OF_NODES
echo MODE: $MODE
echo CASSANDRA_SERVICE: ${CASSANDRA_SERVICE}
export CLASSPATH=$CLASSPATH:/ax/axdb/kubernetes-cassandra.jar
export CASSANDRA_SERVICE=${CASSANDRA_SERVICE}
export MODE=$MODE

exec /axdb/bin/axdb_server $NUMBER_OF_NODES
