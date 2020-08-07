#!/bin/bash
set -eu -o pipefail

# https://github.community/t/what-are-annotations/16173

cat test-results/test.out | while read -r l ; do
  # --- FAIL: TestParse (0.00s)
  if grep -- '--- FAIL: ' <(echo $l) > /dev/null; then
    test=$(echo $l | sed 's/--- FAIL:.\([^ ]*\).*/\1/')
    true > test-results/annotations
  fi

  #     parse_test.go:10: assertion failed: a (string) != b (string)
  if grep -- '_test.go:' <(echo $l) > /dev/null; then
    file=$(echo $l | sed 's/ *\([^ ]*_test.go\).*/\1/')
    line=$(echo $l | sed 's/[^:]*:\([0-9]*\).*/\1/')
    msg=$(echo $l | sed 's/.*: //')
    echo "test=$test; file=$file; line=$line; msg=\"$msg\"" >> test-results/annotations
  fi

  # FAIL	github.com/argoproj/argo/util/intstr	0.089s
  if grep -- '^FAIL.' <(echo $l) > /dev/null; then
    dir=$(echo $l | sed 's/FAIL.\([^ ]*\).*/\1/')

    cat test-results/annotations | while read -r a ; do
      eval $a
      echo "::error file=$dir/$file,line=$line,col=0::$msg"
    done
  fi
done