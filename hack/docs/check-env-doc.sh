#!/usr/bin/env bash

echo "Checking docs/environment-variables.md for completeness..."

# Directories to check. For cmd/, only check Controller, Executor, and Server. The CLI has generated docs
dirs=(./workflow ./persist ./util ./server ./cmd/argo/commands/server.go ./cmd/argoexec ./cmd/workflow-controller)
not_found="false"

function check-used {
  mapfile -t check < <(grep "| \`" < ./docs/environment-variables.md \
    | awk '{gsub(/\`/, "", $2);  print $2; }')

  for x in "${check[@]}"; do
    var="${x%\`}";
    var="${var#\`}";
    if ! grep -qR --exclude="*_test.go" "$var" "${dirs[@]}" ; then
      echo "❌ Documented variable $var in docs/environment-variables.md is not used anywhere" >&2;
      not_found="true";
    fi
  done
}

function check-documented {
  mapfile -t check < <(grep -REh --exclude="*_test.go" "Getenv.*?\(|LookupEnv.*?\(|env.Get*?\(" "${dirs[@]}" \
    | grep -Eo "\"[A-Z_]+?\"" \
    | sort \
    | uniq)

  for x in "${check[@]}"; do
    var="${x%\"}";
    var="${var#\"}";
    if ! grep -q "$var" docs/environment-variables.md; then
      echo "❌ Variable $var not documented in docs/environment-variables.md" >&2;
      not_found="true";
    fi
  done
}

check-used && check-documented;
if [[ "$not_found" == "true" ]]; then
  exit 1;
fi
echo "✅ Success - all environment variables appear to be documented"
