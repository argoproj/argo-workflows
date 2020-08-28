#!/bin/bash
set -eu -o pipefail

now() {
  date +%s
}

t=$(now)
while read -r line; do
  d=$(($(now) - $t))
  if [ $d -gt 0 ]; then
    echo ${d}s "$line"
  else
    echo "    " $line
  fi
  t=$(now)
done
