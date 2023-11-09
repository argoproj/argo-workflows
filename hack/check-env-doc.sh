#!/bin/bash

echo "Checking docs/environment-variables.md for completeness..."

function check-used {
    grep "| \`" < ./docs/environment-variables.md \
      | awk '{gsub(/\`/, "", $2);  print $2; }' \
      | while read -r x; do
        var="${x%\`}";
        var="${var#\`}";
        if ! grep -qR --exclude="*_test.go" "$var" ./cmd/workflow-controller ./workflow ./persist ./util ./server ; then
          echo "❌ Documented variable $var in docs/environment-variables.md is not used anywhere" >&2;
          exit 1;
        fi;
      done
}

function check-documented {
    grep -REh --exclude="*_test.go" "Getenv.*?\(|LookupEnv.*?\(" ./workflow ./persist ./util \
      | grep -Eo "\"[A-Z_]+?\"" \
      | sort \
      | uniq \
      | while read -r x; do
        var="${x%\"}";
        var="${var#\"}";
        if ! grep -q "$var" docs/environment-variables.md; then
          echo "❌ Variable $var not documented in docs/environment-variables.md" >&2;
          exit 1;
        fi;
      done
}

check-used && check-documented && echo "✅ Success - all environment variables appear to be documented"
