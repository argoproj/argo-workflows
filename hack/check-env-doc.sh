#!/bin/bash

pwd
ls -al


echo "Checking env variables doc..."

function check-used {
    grep "| \`" < ./docs/environment-variables.md \
      | awk '{gsub(/\`/, "", $2);  print $2; }' \
      | while read -r x; do
        if ! grep -qR --exclude="*_test.go" "$x" ./workflow ./persist ./util; then
          echo "Documented variable $x is not used anywhere";
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
          echo "Variable $var not documented in docs/environment-variables";
          exit 1;
        fi;
      done
}

check-used && check-documented && echo "Success!"
