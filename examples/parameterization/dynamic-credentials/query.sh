#!/bin/sh
# Only the trusted task process receives dynamic credentials. No credential output.
set -eu
unset VAULT_TOKEN VAULT_ADDR BOOTSTRAP
count=0
while [ "$count" -lt 18 ]; do
  if ! result=$(psql -XAtqc 'SELECT 42' 2>/dev/null); then
    echo credential-query-rejected
    exit 42
  fi
  test "$result" = 42
  count=$((count + 1))
  echo credential-query-ok
  sleep 2
done
echo credential-task-complete
